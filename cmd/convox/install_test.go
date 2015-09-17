package main

import (
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/convox/rack/api/awsutil"
	"github.com/convox/rack/test"
	"github.com/convox/release/version"
)

func TestConvoxInstallSTDINCredentials(t *testing.T) {
	stackId := "arn:aws:cloudformation:us-east-1:123456789:stack/MyStack/aaf549a0-a413-11df-adb3-5081b3858e83"
	cycles := []awsutil.Cycle{
		awsutil.Cycle{
			awsutil.Request{"/", "", "/./"},
			awsutil.Response{200, `<CreateStackResult><StackId>` + stackId + `</StackId></CreateStackResult>`},
		},
		awsutil.Cycle{
			awsutil.Request{"/", "", ""},
			awsutil.Response{200, ""},
		},
	}

	handler := awsutil.NewHandler(cycles)
	s := httptest.NewServer(handler)
	defaults.DefaultConfig.Endpoint = &s.URL

	defer s.Close()

	latest, _ := version.Latest()

	test.Runs(t,
		test.ExecRun{
			Command: "convox install",
			Exit:    0,
			Env:     map[string]string{"AWS_ENDPOINT_URL": s.URL, "AWS_REGION": "test"},
			Stdin:   `{"Credentials":{"AccessKeyId":"FOO","SecretAccessKey":"BAR","Expiration":"2015-09-17T14:09:41Z"}}`,
			Stdout:  Banner + "\nInstalling Convox (" + latest + ")...\n" + stackId + "\n",
		},
	)
}

func TestConvoxInstallFileCredentials(t *testing.T) {

}
