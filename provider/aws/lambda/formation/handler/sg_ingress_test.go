package handler

import (
	"os"
	"testing"
)

func TestSGIngress(t *testing.T) {
	if os.Getenv("MANUAL_TEST") != "true" {
		t.Skip()
	}

	err := sgIngressApply(Request{
		ResourceProperties: map[string]interface{}{
			"SecurityGroupID": "sg-019eb79e77bba7daa",
			"Ips":             "",
			"SgIDs":           "",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSGIngress2(t *testing.T) {
	if os.Getenv("MANUAL_TEST") != "true" {
		t.Skip()
	}

	err := sgIngressApply(Request{
		ResourceProperties: map[string]interface{}{
			"SecurityGroupID": "sg-019eb79e77bba7daa",
			"Ips":             "10.0.0.0/16",
			"SgIDs":           "",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSGIngress3(t *testing.T) {
	if os.Getenv("MANUAL_TEST") != "true" {
		t.Skip()
	}
	err := sgIngressApply(Request{
		ResourceProperties: map[string]interface{}{
			"SecurityGroupID": "sg-019eb79e77bba7daa",
			"Ips":             "10.0.0.0/16",
			"SgIDs":           "sg-0cb8ec0e8c9505ffa", //sg-0e1a179a4c9307a55",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}
