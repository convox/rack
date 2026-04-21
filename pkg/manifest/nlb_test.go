package manifest_test

import (
	"os"
	"strings"
	"testing"

	"github.com/convox/rack/pkg/manifest"
)

// loadBytes is a test helper that loads a manifest from bytes and returns the manifest and error.
func loadBytes(t *testing.T, data []byte) (*manifest.Manifest, error) {
	t.Helper()
	return manifest.Load(data, nil)
}

// loadFixture reads a named testdata file and loads it.
func loadFixture(t *testing.T, name string) (*manifest.Manifest, error) {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return manifest.Load(data, nil)
}

// requireErrContains asserts err is non-nil and its message contains the substring.
func requireErrContains(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected error containing %q, got %q", want, err.Error())
	}
}

func TestManifestLoadNLB(t *testing.T) {
	m, err := loadFixture(t, "nlb.yml")
	if err != nil {
		t.Fatalf("valid nlb manifest failed to load: %v", err)
	}
	s, err := m.Service("api")
	if err != nil {
		t.Fatal(err)
	}
	if len(s.NLB) != 2 {
		t.Fatalf("expected 2 nlb ports, got %d", len(s.NLB))
	}
	if s.NLB[0].Port != 8443 || s.NLB[0].Protocol != "tcp" || s.NLB[0].ContainerPort != 8443 || s.NLB[0].Scheme != "public" {
		t.Errorf("nlb[0] mismatch: %+v", s.NLB[0])
	}
	if s.NLB[1].Scheme != "internal" {
		t.Errorf("nlb[1].Scheme = %q, want internal", s.NLB[1].Scheme)
	}
}

func TestManifestLoadNLBDuplicatePort(t *testing.T) {
	_, err := loadFixture(t, "invalid-nlb-dup-port.yml")
	requireErrContains(t, err, "duplicate nlb port 8443")
}

func TestManifestLoadNLBConflictingContainerPort(t *testing.T) {
	_, err := loadFixture(t, "invalid-nlb-conflicting-container-port.yml")
	requireErrContains(t, err, "conflicting containerPort")
}

func TestManifestLoadNLBBadProtocolUDP(t *testing.T) {
	_, err := loadFixture(t, "invalid-nlb-bad-proto.yml")
	requireErrContains(t, err, "protocol must be tcp or tls")
}

func TestManifestLoadNLBBadProtocolTCPUDP(t *testing.T) {
	_, err := loadFixture(t, "invalid-nlb-bad-tcpudp.yml")
	requireErrContains(t, err, "protocol must be tcp or tls")
}

func TestManifestLoadNLBBadScheme(t *testing.T) {
	_, err := loadFixture(t, "invalid-nlb-bad-scheme.yml")
	requireErrContains(t, err, "scheme must be public or internal")
}

func TestManifestLoadNLBPortZero(t *testing.T) {
	_, err := loadFixture(t, "invalid-nlb-port-zero.yml")
	requireErrContains(t, err, "out of range")
}

func TestManifestLoadNLBPortTooHigh(t *testing.T) {
	_, err := loadFixture(t, "invalid-nlb-port-too-high.yml")
	requireErrContains(t, err, "out of range")
}

func TestManifestLoadNLBAgentConflict(t *testing.T) {
	_, err := loadFixture(t, "invalid-nlb-agent.yml")
	requireErrContains(t, err, "agent mode is incompatible with nlb ports")
}

func TestManifestLoadNLBDefaults(t *testing.T) {
	m, err := loadBytes(t, []byte(`services:
  api:
    image: x
    nlb:
      - port: 8443
`))
	if err != nil {
		t.Fatal(err)
	}
	s, err := m.Service("api")
	if err != nil {
		t.Fatal(err)
	}
	if s.NLB[0].Protocol != "tcp" {
		t.Errorf("default protocol = %q, want tcp", s.NLB[0].Protocol)
	}
	if s.NLB[0].Scheme != "public" {
		t.Errorf("default scheme = %q, want public", s.NLB[0].Scheme)
	}
	if s.NLB[0].ContainerPort != 8443 {
		t.Errorf("default containerPort = %d, want 8443", s.NLB[0].ContainerPort)
	}
}

func TestManifestLoadNLBCaseInsensitive(t *testing.T) {
	m, err := loadBytes(t, []byte(`services:
  api:
    image: x
    nlb:
      - port: 8443
        protocol: TCP
        scheme: Public
`))
	if err != nil {
		t.Fatal(err)
	}
	s, _ := m.Service("api")
	if s.NLB[0].Protocol != "tcp" {
		t.Errorf("protocol should be normalized: got %q", s.NLB[0].Protocol)
	}
	if s.NLB[0].Scheme != "public" {
		t.Errorf("scheme should be normalized: got %q", s.NLB[0].Scheme)
	}
}

func TestManifestLoadNLBDifferentContainerPort(t *testing.T) {
	m, err := loadFixture(t, "nlb-different-container-port.yml")
	if err != nil {
		t.Fatalf("valid manifest failed to load: %v", err)
	}
	s, _ := m.Service("api")
	if s.NLB[0].Port != 8443 || s.NLB[0].ContainerPort != 3000 {
		t.Errorf("expected port=8443, containerPort=3000, got %+v", s.NLB[0])
	}
}

func TestManifestLoadNLBNullAndEmpty(t *testing.T) {
	// nlb: null
	m1, err := loadBytes(t, []byte(`services:
  api:
    image: x
    nlb:
`))
	if err != nil {
		t.Fatal(err)
	}
	s1, _ := m1.Service("api")
	if len(s1.NLB) != 0 {
		t.Errorf("nlb: null should yield empty slice, got %+v", s1.NLB)
	}

	// nlb: []
	m2, err := loadBytes(t, []byte(`services:
  api:
    image: x
    nlb: []
`))
	if err != nil {
		t.Fatal(err)
	}
	s2, _ := m2.Service("api")
	if len(s2.NLB) != 0 {
		t.Errorf("nlb: [] should yield empty slice, got %+v", s2.NLB)
	}
}

func TestManifestLoadNLBDuplicateContainerPort(t *testing.T) {
	data := []byte(`services:
  api:
    image: x
    nlb:
      - port: 8080
        containerPort: 443
      - port: 9090
        containerPort: 443
`)
	if _, err := manifest.Load(data, nil); err == nil {
		t.Fatal("expected error for duplicate containerPort across nlb listeners")
	}
}

func TestManifestLoadNLBCoexistsWithALBPort(t *testing.T) {
	// port:3000/http on ALB + NLB listener on 3000 targeting same container port — allowed
	m, err := loadBytes(t, []byte(`services:
  api:
    image: x
    port: http:3000
    nlb:
      - port: 3000
`))
	if err != nil {
		t.Fatalf("expected ALB+NLB on same port to be allowed: %v", err)
	}
	s, _ := m.Service("api")
	if len(s.NLB) != 1 || s.NLB[0].Port != 3000 {
		t.Errorf("expected 1 NLB port=3000, got %+v", s.NLB)
	}
}

func TestManifestLoadNLBCrossServicePortConflict(t *testing.T) {
	_, err := loadBytes(t, []byte(`services:
  api:
    image: x
    nlb:
      - port: 8443
  web:
    image: x
    nlb:
      - port: 8443
`))
	requireErrContains(t, err, "nlb port 8443 declared by services")
}

func TestManifestLoadNLBTLS(t *testing.T) {
	m, err := loadFixture(t, "nlb-tls.yml")
	if err != nil {
		t.Fatalf("valid tls manifest failed to load: %v", err)
	}
	s, _ := m.Service("api")
	if len(s.NLB) != 1 {
		t.Fatalf("expected 1 nlb port, got %d", len(s.NLB))
	}
	np := s.NLB[0]
	if np.Port != 443 || np.Protocol != "tls" || np.ContainerPort != 8080 || np.Scheme != "public" {
		t.Errorf("nlb[0] mismatch: %+v", np)
	}
	if np.Certificate != "arn:aws:acm:us-east-1:123456789012:certificate/00000000-0000-0000-0000-000000000001" {
		t.Errorf("certificate mismatch: got %q", np.Certificate)
	}
}

func TestManifestLoadNLBTLSWithIAMCert(t *testing.T) {
	m, err := loadFixture(t, "nlb-tls-iam.yml")
	if err != nil {
		t.Fatalf("valid iam tls manifest failed to load: %v", err)
	}
	s, _ := m.Service("api")
	np := s.NLB[0]
	if np.Protocol != "tls" {
		t.Errorf("protocol = %q, want tls", np.Protocol)
	}
	if np.Certificate != "arn:aws:iam::123456789012:server-certificate/my-server-cert" {
		t.Errorf("certificate mismatch: got %q", np.Certificate)
	}
}

func TestManifestLoadNLBTLSInternal(t *testing.T) {
	m, err := loadFixture(t, "nlb-tls-internal.yml")
	if err != nil {
		t.Fatalf("valid internal tls manifest failed to load: %v", err)
	}
	s, _ := m.Service("api")
	np := s.NLB[0]
	if np.Protocol != "tls" {
		t.Errorf("protocol = %q, want tls", np.Protocol)
	}
	if np.Scheme != "internal" {
		t.Errorf("scheme = %q, want internal", np.Scheme)
	}
	if np.Certificate == "" {
		t.Errorf("expected certificate, got empty")
	}
}

func TestManifestLoadNLBMixedTLSAndTCP(t *testing.T) {
	m, err := loadFixture(t, "nlb-tls-mixed.yml")
	if err != nil {
		t.Fatalf("valid mixed manifest failed to load: %v", err)
	}
	s, _ := m.Service("api")
	if len(s.NLB) != 2 {
		t.Fatalf("expected 2 nlb ports, got %d", len(s.NLB))
	}
	if s.NLB[0].Protocol != "tls" || s.NLB[0].Certificate == "" {
		t.Errorf("nlb[0] tls mismatch: %+v", s.NLB[0])
	}
	if s.NLB[1].Protocol != "tcp" || s.NLB[1].Certificate != "" {
		t.Errorf("nlb[1] tcp mismatch: %+v", s.NLB[1])
	}
}

func TestManifestValidateNLBTLSNoCert(t *testing.T) {
	_, err := loadFixture(t, "invalid-nlb-tls-no-cert.yml")
	requireErrContains(t, err, "protocol tls requires a certificate")
	requireErrContains(t, err, "convox certs list")
}

func TestManifestValidateNLBTLSBadARN(t *testing.T) {
	_, err := loadFixture(t, "invalid-nlb-tls-bad-arn.yml")
	requireErrContains(t, err, "must be a full ACM or IAM server-certificate ARN")
}

func TestManifestValidateNLBTCPWithCert(t *testing.T) {
	_, err := loadFixture(t, "invalid-nlb-tcp-with-cert.yml")
	requireErrContains(t, err, "certificate is only valid with protocol: tls")
}

func TestManifestValidateNLBValidatorOrderingProtocolWinsOverCert(t *testing.T) {
	_, err := loadFixture(t, "invalid-nlb-tls-validator-order.yml")
	requireErrContains(t, err, "protocol must be tcp or tls")
	if err != nil && strings.Contains(err.Error(), "certificate is only valid with protocol: tls") {
		t.Errorf("validator ordering broken: got cert-error before protocol-error: %v", err)
	}
}

func TestManifestLoadNLBNewFields(t *testing.T) {
	m, err := loadBytes(t, []byte(`services:
  api:
    image: x
    nlb:
      - port: 8443
        cross_zone: true
        allow_cidr: ["203.0.113.0/24", "198.51.100.5/32"]
        preserve_client_ip: false
`))
	if err != nil {
		t.Fatal(err)
	}
	s, _ := m.Service("api")
	if s.NLB[0].CrossZone == nil || *s.NLB[0].CrossZone != true {
		t.Errorf("CrossZone: want true, got %v", s.NLB[0].CrossZone)
	}
	if len(s.NLB[0].AllowCIDR) != 2 {
		t.Fatalf("AllowCIDR: want 2 entries, got %+v", s.NLB[0].AllowCIDR)
	}
	if s.NLB[0].AllowCIDR[0] != "203.0.113.0/24" {
		t.Errorf("AllowCIDR[0]: got %q", s.NLB[0].AllowCIDR[0])
	}
	if s.NLB[0].PreserveClientIP == nil || *s.NLB[0].PreserveClientIP != false {
		t.Errorf("PreserveClientIP: want false, got %v", s.NLB[0].PreserveClientIP)
	}
}

func TestManifestLoadNLBNewFieldsDefaultsNil(t *testing.T) {
	m, err := loadBytes(t, []byte(`services:
  api:
    image: x
    nlb:
      - port: 8443
`))
	if err != nil {
		t.Fatal(err)
	}
	s, _ := m.Service("api")
	if s.NLB[0].CrossZone != nil {
		t.Errorf("CrossZone: want nil (inherit), got %v", s.NLB[0].CrossZone)
	}
	if len(s.NLB[0].AllowCIDR) != 0 {
		t.Errorf("AllowCIDR: want empty slice, got %+v", s.NLB[0].AllowCIDR)
	}
	if s.NLB[0].PreserveClientIP != nil {
		t.Errorf("PreserveClientIP: want nil (inherit), got %v", s.NLB[0].PreserveClientIP)
	}
}

func TestManifestValidateNLBAllowCIDRMalformed(t *testing.T) {
	_, err := loadBytes(t, []byte(`services:
  api:
    image: x
    nlb:
      - port: 8443
        allow_cidr: ["not-a-cidr"]
`))
	requireErrContains(t, err, `allow_cidr entry "not-a-cidr" is not a valid IPv4 CIDR`)
}

func TestManifestValidateNLBAllowCIDRIPv6Rejected(t *testing.T) {
	_, err := loadBytes(t, []byte(`services:
  api:
    image: x
    nlb:
      - port: 8443
        allow_cidr: ["2001:db8::/32"]
`))
	requireErrContains(t, err, `is not a valid IPv4 CIDR`)
}

func TestManifestValidateNLBAllowCIDRInvalidOctet(t *testing.T) {
	_, err := loadBytes(t, []byte(`services:
  api:
    image: x
    nlb:
      - port: 8443
        allow_cidr: ["256.0.0.0/8"]
`))
	requireErrContains(t, err, `is not a valid IPv4 CIDR`)
}

func TestManifestValidateNLBAllowCIDRInvalidMask(t *testing.T) {
	_, err := loadBytes(t, []byte(`services:
  api:
    image: x
    nlb:
      - port: 8443
        allow_cidr: ["10.0.0.0/33"]
`))
	requireErrContains(t, err, `is not a valid IPv4 CIDR`)
}

func TestManifestValidateNLBAllowCIDRNonCanonical(t *testing.T) {
	_, err := loadBytes(t, []byte(`services:
  api:
    image: x
    nlb:
      - port: 8443
        allow_cidr: ["10.0.0.1/24"]
`))
	requireErrContains(t, err, `not canonical`)
}

func TestManifestValidateNLBAllowCIDRDuplicate(t *testing.T) {
	_, err := loadBytes(t, []byte(`services:
  api:
    image: x
    nlb:
      - port: 8443
        allow_cidr: ["10.0.0.0/24", "10.0.0.0/24"]
`))
	requireErrContains(t, err, `allow_cidr contains duplicate entry: 10.0.0.0/24`)
}

func TestManifestLoadNLBCrossZoneValue(t *testing.T) {
	tru := true
	fals := false
	cases := []struct {
		name string
		np   manifest.ServiceNLBPort
		want string
	}{
		{"nil inherits", manifest.ServiceNLBPort{}, ""},
		{"true", manifest.ServiceNLBPort{CrossZone: &tru}, "true"},
		{"false", manifest.ServiceNLBPort{CrossZone: &fals}, "false"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.np.CrossZoneValue(); got != tc.want {
				t.Errorf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestManifestLoadNLBPreserveClientIPValue(t *testing.T) {
	tru := true
	fals := false
	cases := []struct {
		name   string
		np     manifest.ServiceNLBPort
		pubDef string
		intDef string
		want   string
	}{
		{"nil+public default No", manifest.ServiceNLBPort{Scheme: "public"}, "No", "No", "false"},
		{"nil+public default Yes", manifest.ServiceNLBPort{Scheme: "public"}, "Yes", "No", "true"},
		{"nil+internal default Yes", manifest.ServiceNLBPort{Scheme: "internal"}, "No", "Yes", "true"},
		{"nil+internal default No", manifest.ServiceNLBPort{Scheme: "internal"}, "Yes", "No", "false"},
		{"per-port true wins", manifest.ServiceNLBPort{Scheme: "public", PreserveClientIP: &tru}, "No", "No", "true"},
		{"per-port false wins", manifest.ServiceNLBPort{Scheme: "public", PreserveClientIP: &fals}, "Yes", "Yes", "false"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.np.PreserveClientIPValue(tc.pubDef, tc.intDef); got != tc.want {
				t.Errorf("got %q want %q", got, tc.want)
			}
		})
	}
}
