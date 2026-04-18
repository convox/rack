package aws

import (
	"testing"

	"github.com/convox/rack/pkg/manifest"
)

func TestValidateNLBSchemeMatch_PublicDisabled(t *testing.T) {
	p := &Provider{NLB: false}
	m := &manifest.Manifest{Services: []manifest.Service{{
		Name: "api",
		NLB:  []manifest.ServiceNLBPort{{Port: 8443, Scheme: "public"}},
	}}}
	if err := p.validateNLBSchemeMatch(m); err == nil {
		t.Fatal("expected error when declaring public nlb on rack with NLB=No")
	}
}

func TestValidateNLBSchemeMatch_InternalDisabled(t *testing.T) {
	p := &Provider{NLBInternal: false}
	m := &manifest.Manifest{Services: []manifest.Service{{
		Name: "api",
		NLB:  []manifest.ServiceNLBPort{{Port: 8443, Scheme: "internal"}},
	}}}
	if err := p.validateNLBSchemeMatch(m); err == nil {
		t.Fatal("expected error when declaring internal nlb on rack with NLBInternal=No")
	}
}

func TestValidateNLBSchemeMatch_BothEnabled(t *testing.T) {
	p := &Provider{NLB: true, NLBInternal: true}
	m := &manifest.Manifest{Services: []manifest.Service{{
		Name: "api",
		NLB: []manifest.ServiceNLBPort{
			{Port: 8443, Scheme: "public"},
			{Port: 50051, Scheme: "internal"},
		},
	}}}
	if err := p.validateNLBSchemeMatch(m); err != nil {
		t.Fatalf("expected no error when both NLBs enabled: %v", err)
	}
}

func TestValidateNLBSchemeMatch_NoNLBPorts(t *testing.T) {
	p := &Provider{}
	m := &manifest.Manifest{Services: []manifest.Service{{Name: "api"}}}
	if err := p.validateNLBSchemeMatch(m); err != nil {
		t.Fatalf("service without nlb should not error: %v", err)
	}
}
