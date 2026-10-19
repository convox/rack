package aws

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/private/protocol"
	"github.com/aws/aws-sdk-go/private/protocol/jsonrpc"
)

// The vendored ACM client predates these two operations, so they are sent on it by hand.

type acmCertificateOptions struct {
	ValidationMethod *string
}

type acmUpdateCertificateOptionsInput struct {
	CertificateArn *string
	Options        *acmCertificateOptions
}

type acmListCertificateDomainValidationsInput struct {
	CertificateArn *string
	NextToken      *string
}

type acmListCertificateDomainValidationsOutput struct {
	DomainValidationSummaryList []*acmDomainValidationSummary
	NextToken                   *string
}

type acmDomainValidationSummary struct {
	ActiveValidationConfiguration    *acmValidationConfiguration
	DomainName                       *string
	RequestedValidationConfiguration *acmValidationConfiguration
}

type acmValidationConfiguration struct {
	ValidationChallenge *acmValidationChallenge
	ValidationMethod    *string
	ValidationStatus    *string
}

type acmValidationChallenge struct {
	DnsValidationChallenge *acmDnsValidationChallenge
}

type acmDnsValidationChallenge struct {
	ResourceRecord *acmResourceRecord
}

type acmResourceRecord struct {
	Name  *string
	Type  *string
	Value *string
}

func (c *acmValidationConfiguration) record() (string, string, bool) {
	if c == nil || c.ValidationChallenge == nil || c.ValidationChallenge.DnsValidationChallenge == nil {
		return "", "", false
	}

	rr := c.ValidationChallenge.DnsValidationChallenge.ResourceRecord
	if rr == nil || aws.StringValue(rr.Name) == "" || aws.StringValue(rr.Value) == "" {
		return "", "", false
	}

	return aws.StringValue(rr.Name), aws.StringValue(rr.Value), true
}

func (p *Provider) acmUpdateValidationDNS(ctx context.Context, arn string) error {
	op := &request.Operation{Name: "UpdateCertificateOptions", HTTPMethod: "POST", HTTPPath: "/"}

	in := &acmUpdateCertificateOptionsInput{
		CertificateArn: aws.String(arn),
		Options:        &acmCertificateOptions{ValidationMethod: aws.String("DNS")},
	}

	req := p.acm().NewRequest(op, in, &struct{}{})
	req.Handlers.Unmarshal.Swap(jsonrpc.UnmarshalHandler.Name, protocol.UnmarshalDiscardBodyHandler)
	req.SetContext(ctx)

	return req.Send()
}

func (p *Provider) acmDomainValidations(ctx context.Context, arn string) ([]*acmDomainValidationSummary, error) {
	op := &request.Operation{Name: "ListCertificateDomainValidations", HTTPMethod: "POST", HTTPPath: "/"}

	in := &acmListCertificateDomainValidationsInput{CertificateArn: aws.String(arn)}
	summaries := []*acmDomainValidationSummary{}

	for {
		out := &acmListCertificateDomainValidationsOutput{}

		req := p.acm().NewRequest(op, in, out)
		req.SetContext(ctx)

		if err := req.Send(); err != nil {
			return nil, err
		}

		summaries = append(summaries, out.DomainValidationSummaryList...)

		if aws.StringValue(out.NextToken) == "" {
			return summaries, nil
		}

		in.NextToken = out.NextToken
	}
}
