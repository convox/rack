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
	requireErrContains(t, err, "only tcp protocol is currently supported")
}

func TestManifestLoadNLBBadProtocolTLS(t *testing.T) {
	_, err := loadFixture(t, "invalid-nlb-bad-tls.yml")
	requireErrContains(t, err, "only tcp protocol is currently supported")
}

func TestManifestLoadNLBBadProtocolTCPUDP(t *testing.T) {
	_, err := loadFixture(t, "invalid-nlb-bad-tcpudp.yml")
	requireErrContains(t, err, "only tcp protocol is currently supported")
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
