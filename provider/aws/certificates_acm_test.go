package aws

import (
	"io"
	"net/http"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestACMUpdateValidationDNS(t *testing.T) {
	var target, body string

	p := stubProviderHandler(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		target, body = r.Header.Get("X-Amz-Target"), string(data)

		if body == `{"CertificateArn":"arn:conflict","Options":{"ValidationMethod":"DNS"}}` {
			w.WriteHeader(400)
			_, _ = w.Write([]byte(`{"__type":"ConflictException","message":"in progress"}`))
		}
	}))

	require.NoError(t, p.acmUpdateValidationDNS(p.Context(), "arn:x"))
	assert.Equal(t, "CertificateManager.UpdateCertificateOptions", target)
	assert.Equal(t, `{"CertificateArn":"arn:x","Options":{"ValidationMethod":"DNS"}}`, body)

	err := p.acmUpdateValidationDNS(p.Context(), "arn:conflict")
	assert.Equal(t, acm.ErrCodeConflictException, awsError(err))
}

func TestACMDomainValidations(t *testing.T) {
	var mu sync.Mutex
	bodies := []string{}

	p := stubProviderHandler(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)

		mu.Lock()
		bodies = append(bodies, string(data))
		n := len(bodies)
		mu.Unlock()

		assert.Equal(t, "CertificateManager.ListCertificateDomainValidations", r.Header.Get("X-Amz-Target"))

		if n == 1 {
			_, _ = w.Write([]byte(`{"DomainValidationSummaryList":[{"DomainName":"example.com","ActiveValidationConfiguration":{"ValidationMethod":"EMAIL","ValidationStatus":"SUCCESS"},"RequestedValidationConfiguration":{"ValidationMethod":"DNS","ValidationStatus":"PENDING_VALIDATION","ValidationChallenge":{"DnsValidationChallenge":{"ResourceRecord":{"Name":"_a.example.com.","Type":"CNAME","Value":"_b.x.acm-validations.aws."}}}}}],"NextToken":"page-2"}`))
			return
		}

		_, _ = w.Write([]byte(`{"DomainValidationSummaryList":[{"DomainName":"www.example.com"},null,{"DomainName":"api.example.com","RequestedValidationConfiguration":{"ValidationChallenge":{}}}]}`))
	}))

	vs, err := p.acmDomainValidations(p.Context(), "arn:x")
	require.NoError(t, err)

	assert.Equal(t, []string{`{"CertificateArn":"arn:x"}`, `{"CertificateArn":"arn:x","NextToken":"page-2"}`}, bodies)
	require.Len(t, vs, 4)

	name, value, ok := vs[0].RequestedValidationConfiguration.record()
	assert.True(t, ok)
	assert.Equal(t, "_a.example.com.", name)
	assert.Equal(t, "_b.x.acm-validations.aws.", value)

	for _, v := range vs[1:] {
		if v == nil {
			continue
		}
		_, _, ok := v.RequestedValidationConfiguration.record()
		assert.False(t, ok)
	}
}
