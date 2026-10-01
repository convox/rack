package aws

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/pkg/test/awsutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const acmAllKeyTypes = `{"Includes":{"keyTypes":["RSA_1024","RSA_2048","RSA_3072","RSA_4096","EC_prime256v1","EC_secp384r1","EC_secp521r1"]}}`

func fastCertificateListWait(t *testing.T, timeout time.Duration) {
	tick, wait := certificateListWaitTick, certificateListWaitTimeout

	certificateListWaitTick = time.Millisecond
	certificateListWaitTimeout = timeout

	t.Cleanup(func() {
		certificateListWaitTick = tick
		certificateListWaitTimeout = wait
	})
}

func acmCycle(operation, body string, status int, response string) awsutil.Cycle {
	return awsutil.Cycle{
		Request: awsutil.Request{
			RequestURI: "/",
			Operation:  "CertificateManager." + operation,
			Body:       body,
		},
		Response: awsutil.Response{StatusCode: status, Body: response},
	}
}

func acmListCycle(body string, arns ...string) awsutil.Cycle {
	summaries := []string{}

	for _, arn := range arns {
		summaries = append(summaries, `{"CertificateArn":"`+arn+`"}`)
	}

	return acmCycle("ListCertificates", body, 200, `{"CertificateSummaryList":[`+strings.Join(summaries, ",")+`]}`)
}

func acmTagsCycle(arn, tags string) awsutil.Cycle {
	return acmCycle("ListTagsForCertificate", `{"CertificateArn":"`+arn+`"}`, 200, `{"Tags":[`+tags+`]}`)
}

func acmDescribeCycle(arn, fields string) awsutil.Cycle {
	return acmCycle("DescribeCertificate", `{"CertificateArn":"`+arn+`"}`, 200, `{"Certificate":{"CertificateArn":"`+arn+`","Status":"ISSUED",`+fields+`}}`)
}

var iamEmptyListCycle = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Body:       `Action=ListServerCertificates&Version=2010-05-08`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       `<ListServerCertificatesResponse><ListServerCertificatesResult><ServerCertificateMetadataList/><IsTruncated>false</IsTruncated></ListServerCertificatesResult></ListServerCertificatesResponse>`,
	},
}

func TestCertificateGenerateIdempotencyToken(t *testing.T) {
	fastCertificateListWait(t, 5*time.Second)

	request := func(uuid string) []awsutil.Cycle {
		arn := "arn:aws:acm:us-test-1:123456789012:certificate/" + uuid

		return []awsutil.Cycle{
			{
				Request: awsutil.Request{
					RequestURI: "/",
					Operation:  "CertificateManager.RequestCertificate",
					Body:       `/"IdempotencyToken":"[A-Za-z]{32}"/`,
				},
				Response: awsutil.Response{
					StatusCode: 200,
					Body:       `{"CertificateArn":"` + arn + `"}`,
				},
			},
			acmListCycle(acmAllKeyTypes, arn),
			acmListCycle(acmAllKeyTypes, arn),
		}
	}

	cycles := []awsutil.Cycle{}
	cycles = append(cycles, request("1ad574bd-eeb0-466e-b961-74ec8b405093")...)
	cycles = append(cycles, request("2be685ce-ffc1-577f-ca72-85fd9c516103")...)
	cycles = append(cycles, request("3cf796df-00d2-688a-db83-96ae0d627214")...)

	p, bodies := stubProvider(t, cycles...)

	for _, c := range []struct {
		domains []string
		id      string
	}{
		{[]string{"example.org", "www.example.org"}, "acm-74ec8b405093"},
		{[]string{"example.org"}, "acm-85fd9c516103"},
		{[]string{"example.org"}, "acm-96ae0d627214"},
	} {
		cert, err := p.CertificateGenerate(c.domains)
		require.NoError(t, err)
		assert.Equal(t, c.id, cert.Id)
	}

	require.Len(t, *bodies, 9)

	tokens := map[string]bool{}

	for i, pos := range []int{0, 3, 6} {
		var req struct {
			DomainName              string
			IdempotencyToken        string
			SubjectAlternativeNames []string
		}

		require.NoError(t, json.Unmarshal([]byte((*bodies)[pos]), &req))
		assert.Equal(t, "example.org", req.DomainName)
		assert.Regexp(t, `^[A-Za-z]{32}$`, req.IdempotencyToken)

		if i == 0 {
			assert.Equal(t, []string{"www.example.org"}, req.SubjectAlternativeNames)
		} else {
			assert.Nil(t, req.SubjectAlternativeNames)
		}

		tokens[req.IdempotencyToken] = true
	}

	assert.Len(t, tokens, 3)
}

func TestCertificateGenerateWaitsForList(t *testing.T) {
	fastCertificateListWait(t, 5*time.Second)

	arn := "arn:aws:acm:us-test-1:123456789012:certificate/3cf796df-00d2-688a-db83-96ae0d627214"

	p, bodies := stubProvider(t,
		acmCycle("RequestCertificate", `/"IdempotencyToken":"[A-Za-z]{32}"/`, 200, `{"CertificateArn":"`+arn+`"}`),
		acmListCycle(acmAllKeyTypes, arn),
		acmListCycle(acmAllKeyTypes),
		acmListCycle(acmAllKeyTypes, arn),
		acmListCycle(acmAllKeyTypes, arn),
	)

	cert, err := p.CertificateGenerate([]string{"example.org"})
	require.NoError(t, err)
	assert.Equal(t, "acm-96ae0d627214", cert.Id)
	assert.Len(t, *bodies, 5)
}

func TestCertificateGenerateWaitTimeout(t *testing.T) {
	fastCertificateListWait(t, 50*time.Millisecond)

	arn := "arn:aws:acm:us-test-1:123456789012:certificate/3cf796df-00d2-688a-db83-96ae0d627214"

	var requests, lists atomic.Int32

	p := stubProviderHandler(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("X-Amz-Target") {
		case "CertificateManager.RequestCertificate":
			requests.Add(1)
			_, _ = w.Write([]byte(`{"CertificateArn":"` + arn + `"}`))
		case "CertificateManager.ListCertificates":
			lists.Add(1)
			_, _ = w.Write([]byte(`{"CertificateSummaryList":[]}`))
		default:
			w.WriteHeader(404)
		}
	}))

	cert, err := p.CertificateGenerate([]string{"example.org"})
	require.NoError(t, err)
	assert.Equal(t, "acm-96ae0d627214", cert.Id)
	assert.Equal(t, int32(1), requests.Load())
	assert.Greater(t, lists.Load(), int32(1))
}

func TestCertificateImportWaitsForList(t *testing.T) {
	fastCertificateListWait(t, 5*time.Second)

	arn := "arn:aws:acm:us-test-1:123456789012:certificate/3cf796df-00d2-688a-db83-96ae0d627214"

	p, bodies := stubProvider(t,
		acmCycle("ImportCertificate", `{"Certificate":"cHVi","PrivateKey":"a2V5"}`, 200, `{"CertificateArn":"`+arn+`"}`),
		acmDescribeCycle(arn, `"KeyAlgorithm":"EC-prime256v1","DomainName":"example.org"`),
		acmListCycle(acmAllKeyTypes, arn),
		acmListCycle(acmAllKeyTypes, arn),
	)

	cert, err := p.CertificateCreate("pub", "key", structs.CertificateCreateOptions{})
	require.NoError(t, err)
	assert.Equal(t, "acm-96ae0d627214", cert.Id)
	assert.Equal(t, "example.org", cert.Domain)
	assert.Len(t, *bodies, 4)
}

func TestCertificateListKeyTypes(t *testing.T) {
	ec := "arn:aws:acm:us-test-1:123456789012:certificate/3cf796df-00d2-688a-db83-96ae0d627214"
	bare := "arn:aws:acm:us-test-1:123456789012:certificate/1ad574bd-eeb0-466e-b961-74ec8b405093"

	nextPage := strings.TrimSuffix(acmAllKeyTypes, "}") + `,"NextToken":"page-2"}`

	p, bodies := stubProvider(t,
		iamEmptyListCycle,
		acmCycle("ListCertificates", acmAllKeyTypes, 200, `{"CertificateSummaryList":[{"CertificateArn":"`+ec+`","KeyAlgorithm":"EC-prime256v1"}],"NextToken":"page-2"}`),
		acmListCycle(nextPage, bare),
		acmTagsCycle(ec, `{"Key":"Owner"}`),
		acmDescribeCycle(ec, `"KeyAlgorithm":"EC-prime256v1","DomainName":"example.org","SubjectAlternativeNames":["example.org","www.example.org"]`),
		acmTagsCycle(bare, ``),
		acmDescribeCycle(bare, `"KeyAlgorithm":"RSA-2048"`),
	)

	cs, err := p.CertificateList()
	require.NoError(t, err)
	assert.Len(t, *bodies, 7)

	require.Len(t, cs, 2)
	assert.Equal(t, "acm-96ae0d627214", cs[0].Id)
	assert.Equal(t, "example.org", cs[0].Domain)
	assert.Equal(t, []string{"example.org", "www.example.org"}, cs[0].Domains)
	assert.Equal(t, "acm-74ec8b405093", cs[1].Id)
	assert.Equal(t, "", cs[1].Domain)
}

func TestReleaseCertificates(t *testing.T) {
	current := "arn:aws:acm:us-test-1:123456789012:certificate/1ad574bd-eeb0-466e-b961-74ec8b405093"
	expired := "arn:aws:acm:us-test-1:123456789012:certificate/2be685ce-ffc1-577f-ca72-85fd9c516103"

	p, bodies := stubProvider(t,
		iamEmptyListCycle,
		acmListCycle(`{}`, current, expired),
		acmTagsCycle(current, ``),
		acmDescribeCycle(current, `"DomainName":"example.org","NotAfter":4102444800`),
		acmTagsCycle(expired, ``),
		acmDescribeCycle(expired, `"DomainName":"example.org","NotAfter":946684800`),
	)

	cs, err := p.releaseCertificates()
	require.NoError(t, err)
	assert.Len(t, *bodies, 6)

	require.Len(t, cs, 1)
	assert.Equal(t, current, cs[0].Arn)
}

func TestCertificateDeleteExactId(t *testing.T) {
	first := "arn:aws:acm:us-test-1:123456789012:certificate/1ad574bd-eeb0-466e-b961-74ec8b405093"
	second := "arn:aws:acm:us-test-1:123456789012:certificate/2be685ce-ffc1-577f-ca72-85fd9c516103"
	ec := "arn:aws:acm:us-test-1:123456789012:certificate/3cf796df-00d2-688a-db83-96ae0d627214"
	attached := "arn:aws:acm:us-test-1:123456789012:certificate/4e0807e0-11e3-799b-ec94-a7bf1e738325"

	list := acmListCycle(acmAllKeyTypes, first, second)
	listAll := acmCycle("ListCertificates", acmAllKeyTypes, 200, `{"CertificateSummaryList":[{"CertificateArn":"`+first+`"},{"CertificateArn":"`+second+`"},{"CertificateArn":"`+ec+`","KeyAlgorithm":"EC-prime256v1"},{"CertificateArn":"`+attached+`"}]}`)

	acmDelete := func(arn string) awsutil.Cycle {
		return acmCycle("DeleteCertificate", `{"CertificateArn":"`+arn+`"}`, 200, `{}`)
	}

	acmDeleteGone := acmCycle("DeleteCertificate", `{"CertificateArn":"`+first+`"}`, 400, `{"__type":"ResourceNotFoundException","message":"Could not find certificate"}`)
	acmDeleteInUse := acmCycle("DeleteCertificate", `{"CertificateArn":"`+attached+`"}`, 400, `{"__type":"ResourceInUseException","message":"Certificate is in use"}`)

	iamDelete := awsutil.Cycle{
		Request: awsutil.Request{
			RequestURI: "/",
			Body:       `Action=DeleteServerCertificate&ServerCertificateName=acme-2019&Version=2010-05-08`,
		},
		Response: awsutil.Response{StatusCode: 200, Body: `<DeleteServerCertificateResponse/>`},
	}

	for _, c := range []struct {
		id     string
		cycles []awsutil.Cycle
		err    string
	}{
		{"acm-", []awsutil.Cycle{list}, "certificate not found"},
		{"acm-3", []awsutil.Cycle{list}, "certificate not found"},
		{"acm-b405093", []awsutil.Cycle{list}, "certificate not found"},
		{"acm-85fd9c516103", []awsutil.Cycle{list, acmDelete(second)}, ""},
		{"acm-96ae0d627214", []awsutil.Cycle{listAll, acmDelete(ec)}, ""},
		{"acm-74ec8b405093", []awsutil.Cycle{list, acmDeleteGone}, "certificate not found"},
		{"acm-a7bf1e738325", []awsutil.Cycle{listAll, acmDeleteInUse}, "ResourceInUseException: Certificate is in use"},
		{"acme-2019", []awsutil.Cycle{iamDelete}, ""},
	} {
		t.Run(c.id, func(t *testing.T) {
			p, bodies := stubProvider(t, c.cycles...)

			err := p.CertificateDelete(c.id)
			if c.err == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, c.err)
			}

			assert.Len(t, *bodies, len(c.cycles))
		})
	}
}

func TestCertificateApplyGeneration1(t *testing.T) {
	iamArn := "arn:aws:iam::123456789012:server-certificate/cert-convox-1"
	acmArn := "arn:aws:acm:us-test-1:123456789012:certificate/3cf796df-00d2-688a-db83-96ae0d627214"

	body, _, err := generateSelfSignedCertificate("example.org")
	require.NoError(t, err)

	describeStacks := func(balancer string) awsutil.Cycle {
		return awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Body:       `Action=DescribeStacks&StackName=convox-httpd&Version=2010-05-15`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body: `<DescribeStacksResponse><DescribeStacksResult><Stacks><member>` +
					`<StackName>convox-httpd</StackName><StackStatus>UPDATE_COMPLETE</StackStatus>` +
					`<Parameters>` +
					`<member><ParameterKey>BalancerWebType</ParameterKey><ParameterValue>` + balancer + `</ParameterValue></member>` +
					`<member><ParameterKey>WebPort443Listener</ParameterKey><ParameterValue>32410,` + iamArn + `</ParameterValue></member>` +
					`</Parameters>` +
					`<Tags><member><Key>Name</Key><Value>httpd</Value></member></Tags>` +
					`</member></Stacks></DescribeStacksResult></DescribeStacksResponse>`,
			},
		}
	}

	iamList := []awsutil.Cycle{
		{
			Request: awsutil.Request{
				RequestURI: "/",
				Body:       `Action=ListServerCertificates&Version=2010-05-08`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body: `<ListServerCertificatesResponse><ListServerCertificatesResult><ServerCertificateMetadataList><member>` +
					`<Arn>` + iamArn + `</Arn><ServerCertificateName>cert-convox-1</ServerCertificateName><ServerCertificateId>ASCA1</ServerCertificateId><Path>/</Path><Expiration>2100-01-01T00:00:00Z</Expiration>` +
					`</member></ServerCertificateMetadataList><IsTruncated>false</IsTruncated></ListServerCertificatesResult></ListServerCertificatesResponse>`,
			},
		},
		{
			Request: awsutil.Request{
				RequestURI: "/",
				Body:       `Action=GetServerCertificate&ServerCertificateName=cert-convox-1&Version=2010-05-08`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body: `<GetServerCertificateResponse><GetServerCertificateResult><ServerCertificate>` +
					`<CertificateBody>` + string(body) + `</CertificateBody>` +
					`<ServerCertificateMetadata><Arn>` + iamArn + `</Arn><ServerCertificateName>cert-convox-1</ServerCertificateName><ServerCertificateId>ASCA1</ServerCertificateId><Path>/</Path></ServerCertificateMetadata>` +
					`</ServerCertificate></GetServerCertificateResult></GetServerCertificateResponse>`,
			},
		},
		acmListCycle(acmAllKeyTypes),
	}

	acmList := func(algorithm string) []awsutil.Cycle {
		return []awsutil.Cycle{
			iamEmptyListCycle,
			acmListCycle(acmAllKeyTypes, acmArn),
			acmTagsCycle(acmArn, ``),
			acmDescribeCycle(acmArn, `"KeyAlgorithm":"`+algorithm+`","DomainName":"example.org"`),
		}
	}

	keyAlgorithm := func(algorithm string) []awsutil.Cycle {
		return []awsutil.Cycle{acmDescribeCycle(acmArn, `"KeyAlgorithm":"`+algorithm+`","DomainName":"example.org"`)}
	}

	updateStack := func(balancer, arn string) []awsutil.Cycle {
		return []awsutil.Cycle{
			describeStacks(balancer),
			{
				Request: awsutil.Request{
					RequestURI: "/",
					Body: `Action=UpdateStack&Capabilities.member.1=CAPABILITY_IAM&NotificationARNs.member.1=` +
						`&Parameters.member.1.ParameterKey=BalancerWebType&Parameters.member.1.UsePreviousValue=true` +
						`&Parameters.member.2.ParameterKey=WebPort443Listener&Parameters.member.2.ParameterValue=` + url.QueryEscape("32410,"+arn) +
						`&StackName=convox-httpd&Tags.member.1.Key=Name&Tags.member.1.Value=httpd&UsePreviousTemplate=true&Version=2010-05-15`,
				},
				Response: awsutil.Response{StatusCode: 200, Body: `<UpdateStackResponse><UpdateStackResult><StackId>convox-httpd</StackId></UpdateStackResult></UpdateStackResponse>`},
			},
		}
	}

	cycles := func(balancer string, groups ...[]awsutil.Cycle) []awsutil.Cycle {
		all := []awsutil.Cycle{describeStacks(balancer)}

		for _, g := range groups {
			all = append(all, g...)
		}

		return all
	}

	refused := func(algorithm string) string {
		return "certificate acm-96ae0d627214 is " + algorithm + ", which a Classic Load Balancer does not accept from ACM"
	}

	for _, c := range []struct {
		name   string
		id     string
		cycles []awsutil.Cycle
		err    string
	}{
		{"unknown id", "acm-000000000000", cycles("ELB", acmList("RSA-2048")), "certificate not found"},
		{"EC on ELB", "acm-96ae0d627214", cycles("ELB", acmList("EC-prime256v1"), keyAlgorithm("EC-prime256v1")), refused("EC_prime256v1")},
		{"RSA 3072 on ELB", "acm-96ae0d627214", cycles("ELB", acmList("RSA-3072"), keyAlgorithm("RSA-3072")), refused("RSA_3072")},
		{"RSA 4096 on ELB", "acm-96ae0d627214", cycles("ELB", acmList("RSA-4096"), keyAlgorithm("RSA-4096")), refused("RSA_4096")},
		{"EC on ALB", "acm-96ae0d627214", cycles("ALB", acmList("EC-prime256v1"), updateStack("ALB", acmArn)), ""},
		{"RSA 2048 on ELB", "acm-96ae0d627214", cycles("ELB", acmList("RSA-2048"), keyAlgorithm("RSA-2048"), updateStack("ELB", acmArn)), ""},
		{"IAM on ELB", "cert-convox-1", cycles("ELB", iamList, updateStack("ELB", iamArn)), ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			p, bodies := stubProvider(t, c.cycles...)

			err := p.CertificateApply("httpd", "web", 443, c.id)
			if c.err == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, c.err)
			}

			assert.Len(t, *bodies, len(c.cycles))
		})
	}
}
