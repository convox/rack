package aws

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/convox/rack/pkg/manifest"
)

// nlbCertChecker abstracts the AWS calls that the pre-flight needs. The real
// implementation wraps the provider's existing ACM and IAM clients; tests
// inject a stub.
type nlbCertChecker interface {
	DescribeACM(arn string) (status string, err error)
	GetIAM(name string) error
}

// certPreflight validates all NLB TLS certificates referenced in a manifest
// before a release is promoted. It returns a non-nil error if any referenced
// cert is missing, not ISSUED, in a different region, in a different account,
// or otherwise unreachable. Memoized per unique ARN across services.
func certPreflight(m *manifest.Manifest, checker nlbCertChecker) error {
	arns := map[string]struct{}{}
	for _, s := range m.Services {
		for _, np := range s.NLB {
			if np.Protocol == "tls" && np.Certificate != "" {
				arns[np.Certificate] = struct{}{}
			}
		}
	}
	for arn := range arns {
		if err := checkOneCert(arn, checker); err != nil {
			return err
		}
	}
	return nil
}

func checkOneCert(arn string, checker nlbCertChecker) error {
	switch {
	case strings.Contains(arn, ":acm:"):
		status, err := checker.DescribeACM(arn)
		if err != nil {
			return mapACMError(arn, err)
		}
		if status != "ISSUED" {
			return fmt.Errorf("certificate %s: not usable (status: %s)", arn, status)
		}
		return nil
	case strings.Contains(arn, ":iam:"):
		idx := strings.LastIndex(arn, "/")
		if idx < 0 || idx == len(arn)-1 {
			return fmt.Errorf("certificate %s: malformed IAM server-certificate ARN", arn)
		}
		name := arn[idx+1:]
		if err := checker.GetIAM(name); err != nil {
			return mapIAMError(arn, err)
		}
		return nil
	default:
		return fmt.Errorf("certificate %s: unrecognized ARN shape (expected ACM or IAM server-certificate)", arn)
	}
}

func mapACMError(arn string, err error) error {
	if aerr, ok := err.(awserr.Error); ok {
		switch aerr.Code() {
		case "ResourceNotFoundException":
			return fmt.Errorf("certificate %s: not found in this region (is this cert in another region?)", arn)
		case "AccessDeniedException":
			return fmt.Errorf("certificate %s: access denied (cross-account certificates are not supported)", arn)
		}
	}
	return fmt.Errorf("certificate %s: %s", arn, err)
}

func mapIAMError(arn string, err error) error {
	if aerr, ok := err.(awserr.Error); ok && aerr.Code() == "NoSuchEntity" {
		return fmt.Errorf("certificate %s: IAM server certificate not found", arn)
	}
	return fmt.Errorf("certificate %s: %s", arn, err)
}

// providerCertChecker is the production implementation. It wraps the
// provider's existing ACM + IAM clients.
type providerCertChecker struct {
	p *Provider
}

func (pc *providerCertChecker) DescribeACM(arn string) (string, error) {
	out, err := pc.p.acm().DescribeCertificate(&acm.DescribeCertificateInput{
		CertificateArn: aws.String(arn),
	})
	if err != nil {
		return "", err
	}
	if out.Certificate == nil || out.Certificate.Status == nil {
		return "", fmt.Errorf("acm returned no status")
	}
	return *out.Certificate.Status, nil
}

func (pc *providerCertChecker) GetIAM(name string) error {
	_, err := pc.p.iam().GetServerCertificate(&iam.GetServerCertificateInput{
		ServerCertificateName: aws.String(name),
	})
	return err
}
