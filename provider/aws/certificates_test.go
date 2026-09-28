package aws

import (
	"encoding/json"
	"testing"

	"github.com/convox/rack/pkg/test/awsutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCertificateGenerateIdempotencyToken(t *testing.T) {
	request := func(uuid string) awsutil.Cycle {
		return awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "CertificateManager.RequestCertificate",
				Body:       `/"IdempotencyToken":"[A-Za-z]{32}"/`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"CertificateArn":"arn:aws:acm:us-test-1:123456789012:certificate/` + uuid + `"}`,
			},
		}
	}

	p, bodies := stubProvider(t,
		request("1ad574bd-eeb0-466e-b961-74ec8b405093"),
		request("2be685ce-ffc1-577f-ca72-85fd9c516103"),
		request("3cf796df-00d2-688a-db83-96ae0d627214"),
	)

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

	require.Len(t, *bodies, 3)

	tokens := map[string]bool{}

	for i, b := range *bodies {
		var req struct {
			DomainName              string
			IdempotencyToken        string
			SubjectAlternativeNames []string
		}

		require.NoError(t, json.Unmarshal([]byte(b), &req))
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

func TestCertificateDeleteExactId(t *testing.T) {
	first := "arn:aws:acm:us-test-1:123456789012:certificate/1ad574bd-eeb0-466e-b961-74ec8b405093"
	second := "arn:aws:acm:us-test-1:123456789012:certificate/2be685ce-ffc1-577f-ca72-85fd9c516103"

	list := awsutil.Cycle{
		Request: awsutil.Request{
			RequestURI: "/",
			Operation:  "CertificateManager.ListCertificates",
			Body:       `{}`,
		},
		Response: awsutil.Response{
			StatusCode: 200,
			Body:       `{"CertificateSummaryList":[{"CertificateArn":"` + first + `"},{"CertificateArn":"` + second + `"}]}`,
		},
	}

	acmDelete := awsutil.Cycle{
		Request: awsutil.Request{
			RequestURI: "/",
			Operation:  "CertificateManager.DeleteCertificate",
			Body:       `{"CertificateArn":"` + second + `"}`,
		},
		Response: awsutil.Response{StatusCode: 200, Body: `{}`},
	}

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
		{"acm-85fd9c516103", []awsutil.Cycle{list, acmDelete}, ""},
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
