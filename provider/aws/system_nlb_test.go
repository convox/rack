package aws

import (
	"strings"
	"testing"

	"github.com/convox/rack/pkg/structs"
)

func TestValidateNLBParams_InternalOnlyConflict(t *testing.T) {
	p := &Provider{InternalOnly: true}
	err := p.validateNLBParams(structs.SystemUpdateOptions{Parameters: map[string]string{"NLB": "Yes"}})
	if err == nil {
		t.Fatal("expected error enabling NLB on InternalOnly rack")
	}
}

func TestValidateNLBParams_InternalOnlyCompatible(t *testing.T) {
	p := &Provider{InternalOnly: true, Internal: true}
	err := p.validateNLBParams(structs.SystemUpdateOptions{Parameters: map[string]string{"NLBInternal": "Yes"}})
	if err != nil {
		t.Fatalf("InternalOnly+Internal+NLBInternal should pass: %v", err)
	}
}

func TestValidateNLBParams_InternalRequiredForNLBInternal(t *testing.T) {
	p := &Provider{Internal: false}
	err := p.validateNLBParams(structs.SystemUpdateOptions{Parameters: map[string]string{"NLBInternal": "Yes"}})
	if err == nil {
		t.Fatal("expected error enabling NLBInternal without Internal=Yes")
	}
}

func TestValidateNLBParams_InternalAndNLBInternalSameCall(t *testing.T) {
	p := &Provider{Internal: false}
	err := p.validateNLBParams(structs.SystemUpdateOptions{
		Parameters: map[string]string{"Internal": "Yes", "NLBInternal": "Yes"},
	})
	if err != nil {
		t.Fatalf("expected Internal+NLBInternal together to pass: %v", err)
	}
}

func TestValidateNLBParams_NoNLBParamChange(t *testing.T) {
	p := &Provider{}
	err := p.validateNLBParams(structs.SystemUpdateOptions{Parameters: map[string]string{"Foo": "bar"}})
	if err != nil {
		t.Fatalf("non-NLB param change should not error: %v", err)
	}
}

func TestValidateNLBParams_NilParameters(t *testing.T) {
	p := &Provider{}
	err := p.validateNLBParams(structs.SystemUpdateOptions{})
	if err != nil {
		t.Fatalf("nil parameters should not error: %v", err)
	}
}

func TestValidateNLBParams_AllowCIDRTooMany(t *testing.T) {
	p := &Provider{NLB: true}
	opts := structs.SystemUpdateOptions{Parameters: map[string]string{
		"NLBAllowCIDR": "1.0.0.0/24,2.0.0.0/24,3.0.0.0/24,4.0.0.0/24,5.0.0.0/24,6.0.0.0/24",
	}}
	err := p.validateNLBParams(opts)
	if err == nil || !strings.Contains(err.Error(), "accepts at most 5") {
		t.Fatalf("expected limit error, got %v", err)
	}
}

func TestValidateNLBParams_AllowCIDRMalformed(t *testing.T) {
	p := &Provider{NLB: true}
	opts := structs.SystemUpdateOptions{Parameters: map[string]string{
		"NLBAllowCIDR": "not-a-cidr",
	}}
	err := p.validateNLBParams(opts)
	if err == nil || !strings.Contains(err.Error(), "not a valid IPv4 CIDR") {
		t.Fatalf("expected shape error, got %v", err)
	}
}

func TestValidateNLBParams_AllowCIDRDuplicate(t *testing.T) {
	p := &Provider{NLB: true}
	opts := structs.SystemUpdateOptions{Parameters: map[string]string{
		"NLBAllowCIDR": "10.0.0.0/24,10.0.0.0/24",
	}}
	err := p.validateNLBParams(opts)
	if err == nil || !strings.Contains(err.Error(), "duplicate entry: 10.0.0.0/24") {
		t.Fatalf("expected dedup error, got %v", err)
	}
}

func TestValidateNLBParams_AllowCIDRInternalMirror(t *testing.T) {
	p := &Provider{NLB: true, NLBInternal: true, Internal: true}
	opts := structs.SystemUpdateOptions{Parameters: map[string]string{
		"NLBInternalAllowCIDR": "1.0.0.0/24,1.0.0.0/24",
	}}
	err := p.validateNLBParams(opts)
	if err == nil || !strings.Contains(err.Error(), "NLBInternalAllowCIDR contains duplicate entry") {
		t.Fatalf("expected internal dedup error, got %v", err)
	}
}

func TestValidateNLBParams_AllowCIDREmptyOK(t *testing.T) {
	p := &Provider{NLB: true}
	opts := structs.SystemUpdateOptions{Parameters: map[string]string{
		"NLBAllowCIDR": "",
	}}
	if err := p.validateNLBParams(opts); err != nil {
		t.Fatalf("empty NLBAllowCIDR should pass: %v", err)
	}
}

func TestValidateNLBParams_AllowCIDRTrailingCommaOK(t *testing.T) {
	p := &Provider{NLB: true}
	opts := structs.SystemUpdateOptions{Parameters: map[string]string{
		"NLBAllowCIDR": "10.0.0.0/24,",
	}}
	if err := p.validateNLBParams(opts); err != nil {
		t.Fatalf("trailing comma should be tolerated: %v", err)
	}
}

func TestValidateNLBParams_CustomerSGBlocksPreserveClientIP(t *testing.T) {
	p := &Provider{NLB: true, InstanceSecurityGroup: "sg-customer"}
	opts := structs.SystemUpdateOptions{Parameters: map[string]string{
		"NLBPreserveClientIP": "Yes",
	}}
	err := p.validateNLBParams(opts)
	if err == nil || !strings.Contains(err.Error(), "customer-supplied InstanceSecurityGroup") {
		t.Fatalf("expected customer-SG block, got %v", err)
	}
}

func TestValidateNLBParams_CustomerSGBlocksPreserveClientIPInternal(t *testing.T) {
	p := &Provider{NLB: true, NLBInternal: true, Internal: true, InstanceSecurityGroup: "sg-customer"}
	opts := structs.SystemUpdateOptions{Parameters: map[string]string{
		"NLBInternalPreserveClientIP": "Yes",
	}}
	err := p.validateNLBParams(opts)
	if err == nil || !strings.Contains(err.Error(), "NLBInternalPreserveClientIP on a rack with a customer-supplied") {
		t.Fatalf("expected customer-SG block (internal), got %v", err)
	}
}

func TestValidateNLBParams_BlankInstanceSGAllowsPreserve(t *testing.T) {
	p := &Provider{NLB: true}
	opts := structs.SystemUpdateOptions{Parameters: map[string]string{
		"NLBPreserveClientIP": "Yes",
	}}
	if err := p.validateNLBParams(opts); err != nil {
		t.Fatalf("blank InstanceSecurityGroup should allow preserve_client_ip, got %v", err)
	}
}

func TestValidateNLBParams_InverseInterlockBlocksSettingCustomSG(t *testing.T) {
	// Rack currently has preserve_client_ip=Yes and no customer SG. Operator
	// tries to set a customer SG — must be blocked because the SG substitution
	// would silently break NLB traffic.
	p := &Provider{NLB: true, NLBPreserveClientIP: true}
	opts := structs.SystemUpdateOptions{Parameters: map[string]string{
		"InstanceSecurityGroup": "sg-customer",
	}}
	err := p.validateNLBParams(opts)
	if err == nil || !strings.Contains(err.Error(), "cannot set a customer InstanceSecurityGroup while NLBPreserveClientIP=Yes") {
		t.Fatalf("expected inverse interlock error, got %v", err)
	}
}

func TestValidateNLBParams_InverseInterlockInternal(t *testing.T) {
	p := &Provider{NLB: true, NLBInternal: true, Internal: true, NLBInternalPreserveClientIP: true}
	opts := structs.SystemUpdateOptions{Parameters: map[string]string{
		"InstanceSecurityGroup": "sg-customer",
	}}
	err := p.validateNLBParams(opts)
	if err == nil || !strings.Contains(err.Error(), "cannot set a customer InstanceSecurityGroup while NLBInternalPreserveClientIP=Yes") {
		t.Fatalf("expected inverse interlock (internal) error, got %v", err)
	}
}

func TestValidateNLBParams_InverseInterlockAllowsSimultaneousDisable(t *testing.T) {
	// Operator flips preserve_client_ip=No AND sets custom SG in the same call.
	// Must be allowed — at the end of the update, preserve_client_ip is off and
	// the customer-SG caveat doesn't apply.
	p := &Provider{NLB: true, NLBPreserveClientIP: true}
	opts := structs.SystemUpdateOptions{Parameters: map[string]string{
		"InstanceSecurityGroup": "sg-customer",
		"NLBPreserveClientIP":   "No",
	}}
	if err := p.validateNLBParams(opts); err != nil {
		t.Fatalf("same-call disable should be allowed: %v", err)
	}
}

func TestValidateNLBParams_InverseInterlockAllowsBlankPreserve(t *testing.T) {
	// Rack with no preserve_client_ip and no custom SG; operator adds custom SG.
	// Should pass — no preserve_client_ip in force to worry about.
	p := &Provider{NLB: true}
	opts := structs.SystemUpdateOptions{Parameters: map[string]string{
		"InstanceSecurityGroup": "sg-customer",
	}}
	if err := p.validateNLBParams(opts); err != nil {
		t.Fatalf("custom SG on rack without preserve_client_ip should pass: %v", err)
	}
}

func TestValidateNLBParams_AllowCIDRLeadingSpaceRejected(t *testing.T) {
	p := &Provider{NLB: true}
	opts := structs.SystemUpdateOptions{Parameters: map[string]string{
		"NLBAllowCIDR": "10.0.0.0/24, 10.0.0.1/32",
	}}
	err := p.validateNLBParams(opts)
	if err == nil || !strings.Contains(err.Error(), "leading or trailing whitespace") {
		t.Fatalf("expected whitespace rejection, got %v", err)
	}
}

func TestValidateNLBParams_AllowCIDRInvalidOctet(t *testing.T) {
	p := &Provider{NLB: true}
	opts := structs.SystemUpdateOptions{Parameters: map[string]string{
		"NLBAllowCIDR": "256.0.0.0/8",
	}}
	err := p.validateNLBParams(opts)
	if err == nil || !strings.Contains(err.Error(), "not a valid IPv4 CIDR") {
		t.Fatalf("expected invalid-octet rejection, got %v", err)
	}
}

func TestValidateNLBParams_AllowCIDRInvalidMask(t *testing.T) {
	p := &Provider{NLB: true}
	opts := structs.SystemUpdateOptions{Parameters: map[string]string{
		"NLBAllowCIDR": "10.0.0.0/33",
	}}
	err := p.validateNLBParams(opts)
	if err == nil || !strings.Contains(err.Error(), "not a valid IPv4 CIDR") {
		t.Fatalf("expected invalid-mask rejection, got %v", err)
	}
}

func TestValidateNLBParams_AllowCIDRNonCanonicalRejected(t *testing.T) {
	p := &Provider{NLB: true}
	opts := structs.SystemUpdateOptions{Parameters: map[string]string{
		"NLBAllowCIDR": "10.0.0.1/24",
	}}
	err := p.validateNLBParams(opts)
	if err == nil || !strings.Contains(err.Error(), "not canonical") {
		t.Fatalf("expected non-canonical rejection, got %v", err)
	}
}
