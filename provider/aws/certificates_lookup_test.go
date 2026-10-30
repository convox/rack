package aws

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/convox/rack/pkg/cache"
	"github.com/convox/rack/pkg/test/awsutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const acmIssued = `{"CertificateStatuses":["ISSUED"]}`

func TestCertificateListSummary(t *testing.T) {
	full := "arn:aws:acm:us-test-1:123456789012:certificate/1ad574bd-eeb0-466e-b961-74ec8b405093"
	more := "arn:aws:acm:us-test-1:123456789012:certificate/2be685ce-ffc1-577f-ca72-85fd9c516103"
	partial := "arn:aws:acm:us-test-1:123456789012:certificate/3cf796df-00d2-688a-db83-96ae0d627214"

	page := `{"CertificateSummaryList":[` +
		`{"CertificateArn":"` + full + `","DomainName":"example.org","SubjectAlternativeNameSummaries":["www.example.org","example.org"],"NotAfter":4102444800},` +
		`{"CertificateArn":"` + more + `","DomainName":"example.com","SubjectAlternativeNameSummaries":["example.com"],"HasAdditionalSubjectAlternativeNames":true,"NotAfter":4102444800},` +
		`{"CertificateArn":"` + partial + `","DomainName":"example.net","SubjectAlternativeNameSummaries":["www.example.net"],"NotAfter":4102444800}` +
		`]}`

	p, bodies := stubProvider(t,
		iamEmptyListCycle,
		acmCycle("ListCertificates", acmIssued, 200, page),
		acmTagsCycle(full, ``),
		acmTagsCycle(more, ``),
		acmDescribeCycle(more, `"DomainName":"example.com","SubjectAlternativeNames":["example.com","a.example.com"],"NotAfter":4102444800`),
		acmTagsCycle(partial, ``),
		acmDescribeCycle(partial, `"DomainName":"example.net","SubjectAlternativeNames":["example.net","www.example.net"],"NotAfter":4102444800`),
		acmDescribeCycle(full, `"DomainName":"example.org","SubjectAlternativeNames":["www.example.org","example.org"],"NotAfter":4102444800`),
	)

	cs, err := p.certificateList(nil)
	require.NoError(t, err)
	assert.Len(t, *bodies, 7)

	require.Len(t, cs, 3)
	assert.Equal(t, []string{full, more, partial}, []string{cs[0].Arn, cs[1].Arn, cs[2].Arn})
	assert.Equal(t, []string{"example.com", "a.example.com"}, cs[1].Domains)
	assert.Equal(t, []string{"example.net", "www.example.net"}, cs[2].Domains)

	described, err := p.certificateGetACM(full)
	require.NoError(t, err)
	assert.Equal(t, *described, cs[0])
}

func TestCertificateListMemo(t *testing.T) {
	body, _, err := generateSelfSignedCertificate("example.org")
	require.NoError(t, err)

	first := "arn:aws:acm:us-test-1:123456789012:certificate/memo-1ad574bd"
	second := "arn:aws:acm:us-test-1:123456789012:certificate/memo-2be685ce"
	iamArn := "arn:aws:iam::123456789012:server-certificate/memo-cert"

	var listed atomic.Value
	listed.Store(first)

	counts := map[string]*int32{}
	for _, op := range []string{"ListServerCertificates", "GetServerCertificate", "ListCertificates", "ListTagsForCertificate", "DescribeCertificate"} {
		counts[op] = new(int32)
	}

	p := cachingProvider(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		op := strings.TrimPrefix(r.Header.Get("X-Amz-Target"), "CertificateManager.")
		if op == "" {
			op = strings.TrimPrefix(strings.Split(string(b), "&")[0], "Action=")
		}
		atomic.AddInt32(counts[op], 1)

		switch op {
		case "ListServerCertificates":
			_, _ = fmt.Fprint(w, `<ListServerCertificatesResponse><ListServerCertificatesResult><ServerCertificateMetadataList><member><Arn>`+iamArn+`</Arn><ServerCertificateName>memo-cert</ServerCertificateName><ServerCertificateId>ASCAMEMO</ServerCertificateId><Path>/</Path><Expiration>2100-01-01T00:00:00Z</Expiration></member></ServerCertificateMetadataList><IsTruncated>false</IsTruncated></ListServerCertificatesResult></ListServerCertificatesResponse>`)
		case "GetServerCertificate":
			_, _ = fmt.Fprint(w, `<GetServerCertificateResponse><GetServerCertificateResult><ServerCertificate><CertificateBody>`+string(body)+`</CertificateBody><ServerCertificateMetadata><Arn>`+iamArn+`</Arn><ServerCertificateName>memo-cert</ServerCertificateName><ServerCertificateId>ASCAMEMO</ServerCertificateId><Path>/</Path></ServerCertificateMetadata></ServerCertificate></GetServerCertificateResult></GetServerCertificateResponse>`)
		case "ListCertificates":
			summaries := []string{}
			arns, _ := listed.Load().(string)
			for _, arn := range strings.Split(arns, ",") {
				summaries = append(summaries, `{"CertificateArn":"`+arn+`","DomainName":"example.org","SubjectAlternativeNameSummaries":["example.org"],"NotAfter":4102444800}`)
			}
			_, _ = fmt.Fprint(w, `{"CertificateSummaryList":[`+strings.Join(summaries, ",")+`]}`)
		case "ListTagsForCertificate":
			_, _ = fmt.Fprint(w, `{"Tags":[]}`)
		default:
			w.WriteHeader(400)
		}
	})

	t.Cleanup(func() {
		require.NoError(t, cache.Clear("certificateUsable", first))
		require.NoError(t, cache.Clear("certificateUsable", second))
		require.NoError(t, cache.Clear("iamServerCertificate", "ASCAMEMO"))
	})

	for i := 0; i < 2; i++ {
		cs, err := p.certificateList(nil)
		require.NoError(t, err)
		require.Len(t, cs, 2)
	}

	assert.EqualValues(t, 2, *counts["ListServerCertificates"])
	assert.EqualValues(t, 2, *counts["ListCertificates"])
	assert.EqualValues(t, 1, *counts["GetServerCertificate"])
	assert.EqualValues(t, 1, *counts["ListTagsForCertificate"])
	assert.EqualValues(t, 0, *counts["DescribeCertificate"])

	listed.Store(first + "," + second)

	cs, err := p.certificateList(nil)
	require.NoError(t, err)
	require.Len(t, cs, 3)
	assert.EqualValues(t, 2, *counts["ListTagsForCertificate"])
}

func TestCertificateID(t *testing.T) {
	issued := "arn:aws:acm:us-test-1:123456789012:certificate/1ad574bd-eeb0-466e-b961-74ec8b405093"
	app := "arn:aws:acm:us-test-1:123456789012:certificate/2be685ce-ffc1-577f-ca72-85fd9c516103"
	gone := "arn:aws:acm:us-test-1:123456789012:certificate/3cf796df-00d2-688a-db83-96ae0d627214"
	iamArn := "arn:aws:iam::123456789012:server-certificate/cloudfront/site-cert"
	iamGone := "arn:aws:iam::123456789012:server-certificate/old-cert"

	getServerCertificate := func(name string, status int, response string) awsutil.Cycle {
		return awsutil.Cycle{
			Request:  awsutil.Request{RequestURI: "/", Body: `Action=GetServerCertificate&ServerCertificateName=` + name + `&Version=2010-05-08`},
			Response: awsutil.Response{StatusCode: status, Body: response},
		}
	}

	p, bodies := stubProvider(t,
		acmTagsCycle(issued, ``),
		acmDescribeCycle(issued, `"DomainName":"example.org"`),
		acmTagsCycle(app, `{"Key":"System","Value":"convox"},{"Key":"Type","Value":"app"}`),
		acmTagsCycle(gone, ``),
		acmCycle("DescribeCertificate", `{"CertificateArn":"`+gone+`"}`, 400, `{"__type":"ResourceNotFoundException","message":"x"}`),
		getServerCertificate("site-cert", 200, `<GetServerCertificateResponse><GetServerCertificateResult><ServerCertificate><CertificateBody>x</CertificateBody><ServerCertificateMetadata><Arn>`+iamArn+`</Arn><ServerCertificateName>site-cert</ServerCertificateName><ServerCertificateId>ASCA2</ServerCertificateId><Path>/cloudfront/</Path></ServerCertificateMetadata></ServerCertificate></GetServerCertificateResult></GetServerCertificateResponse>`),
		getServerCertificate("old-cert", 404, `<ErrorResponse><Error><Type>Sender</Type><Code>NoSuchEntity</Code><Message>x</Message></Error></ErrorResponse>`),
	)

	tests := []struct {
		arn string
		id  string
	}{
		{issued, "acm-74ec8b405093"},
		{app, ""},
		{gone, ""},
		{iamArn, "site-cert"},
		{iamGone, ""},
		{"", ""},
		{"not-an-arn", ""},
	}

	for _, tt := range tests {
		id, err := p.certificateID(tt.arn)
		require.NoError(t, err, tt.arn)
		assert.Equal(t, tt.id, id, tt.arn)
	}

	assert.Len(t, *bodies, 7)
}

func TestCertificateListIgnoresRequestContext(t *testing.T) {
	p, _ := stubProvider(t,
		iamEmptyListCycle,
		acmCycle("ListCertificates", acmIssued, 200, `{"CertificateSummaryList":[]}`),
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cp, ok := p.WithContext(ctx).(*Provider)
	require.True(t, ok)

	_, err := cp.certificateList(nil)
	require.NoError(t, err)
}
