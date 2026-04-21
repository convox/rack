package aws

import (
	"strings"
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

func TestValidateNLBSchemeMatch_CustomerSGBlocksPreserveClientIP(t *testing.T) {
	tru := true
	p := &Provider{NLB: true, NLBInternal: true, InstanceSecurityGroup: "sg-customer"}
	m := &manifest.Manifest{Services: []manifest.Service{{
		Name: "api",
		NLB: []manifest.ServiceNLBPort{
			{Port: 8443, Scheme: "public", PreserveClientIP: &tru},
		},
	}}}
	err := p.validateNLBSchemeMatch(m)
	if err == nil || !strings.Contains(err.Error(), "customer-supplied InstanceSecurityGroup") {
		t.Fatalf("expected customer-SG block on release promote, got %v", err)
	}
}

func TestValidateNLBSchemeMatch_CustomerSGAllowsPreserveFalse(t *testing.T) {
	fals := false
	p := &Provider{NLB: true, InstanceSecurityGroup: "sg-customer"}
	m := &manifest.Manifest{Services: []manifest.Service{{
		Name: "api",
		NLB: []manifest.ServiceNLBPort{
			{Port: 8443, Scheme: "public", PreserveClientIP: &fals},
		},
	}}}
	if err := p.validateNLBSchemeMatch(m); err != nil {
		t.Fatalf("customer SG + preserve_client_ip=false should pass: %v", err)
	}
}

func TestValidateNLBSchemeMatch_CustomerSGAllowsPreserveNil(t *testing.T) {
	p := &Provider{NLB: true, InstanceSecurityGroup: "sg-customer"}
	m := &manifest.Manifest{Services: []manifest.Service{{
		Name: "api",
		NLB: []manifest.ServiceNLBPort{
			{Port: 8443, Scheme: "public"},
		},
	}}}
	if err := p.validateNLBSchemeMatch(m); err != nil {
		t.Fatalf("customer SG + no per-port override should pass (inherits rack default): %v", err)
	}
}

func TestValidateNLBSchemeMatch_BlankInstanceSGAllowsPreserveTrue(t *testing.T) {
	tru := true
	p := &Provider{NLB: true}
	m := &manifest.Manifest{Services: []manifest.Service{{
		Name: "api",
		NLB: []manifest.ServiceNLBPort{
			{Port: 8443, Scheme: "public", PreserveClientIP: &tru},
		},
	}}}
	if err := p.validateNLBSchemeMatch(m); err != nil {
		t.Fatalf("blank InstanceSecurityGroup + preserve=true should pass: %v", err)
	}
}
