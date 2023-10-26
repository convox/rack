package aws

import (
	"os"
	"testing"
)

func TestTry(t *testing.T) {
	t.Skip()

	p := &Provider{
		Region:              os.Getenv("AWS_REGION"),
		Private:             false,
		WhiteListSpecified:  true,
		ApiBalancerSecurity: "sg-0042ec8049da65c75",
		Rack:                "v2-rack219",
	}

	err := p.SyncInstancesIpInSecurityGroup()
	if err != nil {
		t.Fatal(err)
	}
}
