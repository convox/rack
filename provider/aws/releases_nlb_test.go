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

func TestValidateNLBSchemeMatch_CustomSGPreserveClientIPNeedsRule(t *testing.T) {
	tru := true
	p, bodies := nlbStubProvider(t, nlbRack{nlb: true, nlbInternal: true, sg: "sg-custom"},
		cycleNLBRackOutput("NLBSecurityGroup", "sg-nlb"),
		cycleNLBSecurityGroup(""),
	)
	m := &manifest.Manifest{Services: []manifest.Service{{
		Name: "api",
		NLB: []manifest.ServiceNLBPort{
			{Port: 8443, Scheme: "public", PreserveClientIP: &tru},
		},
	}}}
	err := p.validateNLBSchemeMatch(m)
	if err == nil || !strings.Contains(err.Error(), "service api nlb port 8443: preserve client IP on the public NLB needs an ingress rule on InstanceSecurityGroup sg-custom allowing all traffic from the NLB security group sg-nlb") {
		t.Fatalf("expected missing-rule error on release promote, got %v", err)
	}
	if len(*bodies) != 2 {
		t.Fatalf("expected 2 AWS calls, got %d", len(*bodies))
	}
}

func TestValidateNLBSchemeMatch_CustomSGPreserveClientIPWithRule(t *testing.T) {
	tru := true
	p, bodies := nlbStubProvider(t, nlbRack{nlb: true, nlbInternal: true, sg: "sg-custom"},
		cycleNLBRackOutput("NLBSecurityGroup", "sg-nlb"),
		cycleNLBSecurityGroup(nlbRule("ipPermissions", "-1", "sg-nlb")),
		cycleNLBRackOutput("NLBInternalSecurityGroup", "sg-nlbi"),
		cycleNLBSecurityGroup(nlbRule("ipPermissions", "-1", "sg-nlbi")),
	)
	m := &manifest.Manifest{Services: []manifest.Service{
		{
			Name: "api",
			NLB: []manifest.ServiceNLBPort{
				{Port: 8443, Scheme: "public", PreserveClientIP: &tru},
				{Port: 50051, Scheme: "internal", PreserveClientIP: &tru},
			},
		},
		{
			Name: "worker",
			NLB:  []manifest.ServiceNLBPort{{Port: 9000, Scheme: "public", PreserveClientIP: &tru}},
		},
	}}
	if err := p.validateNLBSchemeMatch(m); err != nil {
		t.Fatalf("custom SG with the NLB rule should pass: %v", err)
	}
	if len(*bodies) != 4 {
		t.Fatalf("expected one rule check per scheme (4 AWS calls), got %d", len(*bodies))
	}
}

func TestValidateNLBSchemeMatch_CustomSGAllowsPreserveFalse(t *testing.T) {
	fals := false
	p, bodies := nlbStubProvider(t, nlbRack{nlb: true, sg: "sg-custom"})
	m := &manifest.Manifest{Services: []manifest.Service{{
		Name: "api",
		NLB: []manifest.ServiceNLBPort{
			{Port: 8443, Scheme: "public", PreserveClientIP: &fals},
		},
	}}}
	if err := p.validateNLBSchemeMatch(m); err != nil {
		t.Fatalf("custom SG + preserve_client_ip=false should pass: %v", err)
	}
	if len(*bodies) != 0 {
		t.Fatalf("expected no AWS calls, got %d", len(*bodies))
	}
}

func TestValidateNLBSchemeMatch_CustomSGAllowsPreserveNil(t *testing.T) {
	p, bodies := nlbStubProvider(t, nlbRack{nlb: true, sg: "sg-custom", preserve: true})
	m := &manifest.Manifest{Services: []manifest.Service{{
		Name: "api",
		NLB: []manifest.ServiceNLBPort{
			{Port: 8443, Scheme: "public"},
		},
	}}}
	if err := p.validateNLBSchemeMatch(m); err != nil {
		t.Fatalf("custom SG + no per-port override should pass (inherits rack default): %v", err)
	}
	if len(*bodies) != 0 {
		t.Fatalf("expected no AWS calls, got %d", len(*bodies))
	}
}

func TestValidateNLBSchemeMatch_BlankInstanceSGAllowsPreserveTrue(t *testing.T) {
	tru := true
	p, bodies := nlbStubProvider(t, nlbRack{nlb: true})
	m := &manifest.Manifest{Services: []manifest.Service{{
		Name: "api",
		NLB: []manifest.ServiceNLBPort{
			{Port: 8443, Scheme: "public", PreserveClientIP: &tru},
		},
	}}}
	if err := p.validateNLBSchemeMatch(m); err != nil {
		t.Fatalf("blank InstanceSecurityGroup + preserve=true should pass: %v", err)
	}
	if len(*bodies) != 0 {
		t.Fatalf("expected no AWS calls, got %d", len(*bodies))
	}
}
