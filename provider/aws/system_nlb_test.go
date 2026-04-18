package aws

import (
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
