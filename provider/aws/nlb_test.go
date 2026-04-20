package aws

import (
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/convox/rack/pkg/manifest"
)

// stubCertChecker implements nlbCertChecker with programmable responses and
// call-count tracking. Default behavior: unknown ACM ARN → ResourceNotFoundException,
// unknown IAM name → NoSuchEntity.
type stubCertChecker struct {
	acmResponses map[string]acmStub
	iamResponses map[string]error
	acmCalls     map[string]int
	iamCalls     map[string]int
}

type acmStub struct {
	status string
	err    error
}

func newStubCertChecker() *stubCertChecker {
	return &stubCertChecker{
		acmResponses: map[string]acmStub{},
		iamResponses: map[string]error{},
		acmCalls:     map[string]int{},
		iamCalls:     map[string]int{},
	}
}

func (s *stubCertChecker) DescribeACM(arn string) (string, error) {
	s.acmCalls[arn]++
	r, ok := s.acmResponses[arn]
	if !ok {
		return "", awsErr("ResourceNotFoundException", "no such certificate")
	}
	return r.status, r.err
}

func (s *stubCertChecker) GetIAM(name string) error {
	s.iamCalls[name]++
	if e, ok := s.iamResponses[name]; ok {
		return e
	}
	return awsErr("NoSuchEntity", "server certificate not found")
}

func awsErr(code, msg string) error {
	return awserr.New(code, msg, nil)
}

func manifestWithOneTLS(arn string) *manifest.Manifest {
	return &manifest.Manifest{
		Services: manifest.Services{
			{
				Name: "api",
				NLB:  []manifest.ServiceNLBPort{{Port: 443, Protocol: "tls", Certificate: arn}},
			},
		},
	}
}

func manifestWithTwoServicesSameCert(arn string) *manifest.Manifest {
	return &manifest.Manifest{
		Services: manifest.Services{
			{Name: "api", NLB: []manifest.ServiceNLBPort{{Port: 443, Protocol: "tls", Certificate: arn}}},
			{Name: "web", NLB: []manifest.ServiceNLBPort{{Port: 8443, Protocol: "tls", Certificate: arn}}},
		},
	}
}

const (
	goodACMArn  = "arn:aws:acm:us-east-1:123456789012:certificate/aaaa-1111"
	otherACMArn = "arn:aws:acm:us-east-1:123456789012:certificate/bbbb-2222"
	goodIAMArn  = "arn:aws:iam::123456789012:server-certificate/my-cert"
)

func TestCertPreflightACMIssuedPasses(t *testing.T) {
	ch := newStubCertChecker()
	ch.acmResponses[goodACMArn] = acmStub{status: "ISSUED"}
	if err := certPreflight(manifestWithOneTLS(goodACMArn), ch); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestCertPreflightACMNonIssuedFails(t *testing.T) {
	statuses := []string{
		"PENDING_VALIDATION", "INACTIVE", "EXPIRED",
		"VALIDATION_TIMED_OUT", "REVOKED", "FAILED",
		"SOMETHING_NEW_FROM_AWS", // reject-by-default guarantee
	}
	for _, status := range statuses {
		t.Run(status, func(t *testing.T) {
			ch := newStubCertChecker()
			ch.acmResponses[goodACMArn] = acmStub{status: status}
			err := certPreflight(manifestWithOneTLS(goodACMArn), ch)
			if err == nil {
				t.Fatalf("expected error for status %s, got nil", status)
			}
			if !strings.Contains(err.Error(), "not usable") || !strings.Contains(err.Error(), status) {
				t.Errorf("error must mention 'not usable' and %q, got %q", status, err.Error())
			}
		})
	}
}

func TestCertPreflightACMCrossRegion(t *testing.T) {
	ch := newStubCertChecker()
	// default stub behavior for unknown ARN: ResourceNotFoundException.
	err := certPreflight(manifestWithOneTLS(otherACMArn), ch)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not found in this region") {
		t.Errorf("expected cross-region message, got %q", err.Error())
	}
}

func TestCertPreflightACMCrossAccount(t *testing.T) {
	ch := newStubCertChecker()
	ch.acmResponses[goodACMArn] = acmStub{err: awsErr("AccessDeniedException", "not your cert")}
	err := certPreflight(manifestWithOneTLS(goodACMArn), ch)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "access denied") || !strings.Contains(err.Error(), "cross-account") {
		t.Errorf("expected cross-account message, got %q", err.Error())
	}
}

func TestCertPreflightIAMFound(t *testing.T) {
	ch := newStubCertChecker()
	ch.iamResponses["my-cert"] = nil
	if err := certPreflight(manifestWithOneTLS(goodIAMArn), ch); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestCertPreflightIAMNotFound(t *testing.T) {
	ch := newStubCertChecker()
	err := certPreflight(manifestWithOneTLS(goodIAMArn), ch)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "IAM server certificate not found") {
		t.Errorf("expected IAM-not-found message, got %q", err.Error())
	}
}

func TestCertPreflightPrefixDispatchACMOnly(t *testing.T) {
	ch := newStubCertChecker()
	ch.acmResponses[goodACMArn] = acmStub{status: "ISSUED"}
	_ = certPreflight(manifestWithOneTLS(goodACMArn), ch)
	if ch.acmCalls[goodACMArn] != 1 {
		t.Errorf("ACM should be called exactly once for ACM ARN, got %d", ch.acmCalls[goodACMArn])
	}
	if len(ch.iamCalls) != 0 {
		t.Errorf("IAM should never be called for ACM ARN, got %v", ch.iamCalls)
	}
}

func TestCertPreflightPrefixDispatchIAMOnly(t *testing.T) {
	ch := newStubCertChecker()
	ch.iamResponses["my-cert"] = nil
	_ = certPreflight(manifestWithOneTLS(goodIAMArn), ch)
	if ch.iamCalls["my-cert"] != 1 {
		t.Errorf("IAM should be called exactly once for IAM ARN, got %d", ch.iamCalls["my-cert"])
	}
	if len(ch.acmCalls) != 0 {
		t.Errorf("ACM should never be called for IAM ARN, got %v", ch.acmCalls)
	}
}

func TestCertPreflightMemoization(t *testing.T) {
	ch := newStubCertChecker()
	ch.acmResponses[goodACMArn] = acmStub{status: "ISSUED"}
	if err := certPreflight(manifestWithTwoServicesSameCert(goodACMArn), ch); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if ch.acmCalls[goodACMArn] != 1 {
		t.Errorf("two services sharing one cert should produce one call, got %d", ch.acmCalls[goodACMArn])
	}
}

func TestCertPreflightAnyAWSError(t *testing.T) {
	ch := newStubCertChecker()
	ch.acmResponses[goodACMArn] = acmStub{err: awsErr("Throttling", "rate limited")}
	err := certPreflight(manifestWithOneTLS(goodACMArn), ch)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), fmt.Sprintf("certificate %s:", goodACMArn)) {
		t.Errorf("expected 'certificate %s:' prefix, got %q", goodACMArn, err.Error())
	}
}

func TestCertPreflightSkipsNonTLSEntries(t *testing.T) {
	ch := newStubCertChecker()
	m := &manifest.Manifest{
		Services: manifest.Services{
			{Name: "api", NLB: []manifest.ServiceNLBPort{{Port: 443, Protocol: "tcp"}}},
		},
	}
	if err := certPreflight(m, ch); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if len(ch.acmCalls) != 0 || len(ch.iamCalls) != 0 {
		t.Errorf("should not call AWS for non-TLS entries, got acm=%v iam=%v", ch.acmCalls, ch.iamCalls)
	}
}
