package main

import (
	"net/http/httptest"
	"os"
	"testing"

	"github.com/convox/rack/api/awsutil"
	"github.com/convox/rack/test"
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
	defer s.Close()

	os.Setenv("AWS_ENDPOINT", s.URL)

	test.Runs(t,
		test.ExecRun{
			Command: "convox install",
			Exit:    1,
			Env:     map[string]string{"AWS_ENDPOINT_URL": s.URL, "AWS_REGION": "test"},
			Stdin:   `{"Credentials":{"AccessKeyId":"FOO","SecretAccessKey":"BAR","Expiration":"2015-09-17T14:09:41Z"}}`,
			Stdout:  Banner + "\n",
		},
	)
}

func TestConvoxInstallValidateStackName(t *testing.T) {
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
	defer s.Close()

	os.Setenv("AWS_ENDPOINT", s.URL)

	test.Runs(t,
		test.ExecRun{
			Command: "convox install --stack-name valid",
			Exit:    1,
			Env:     map[string]string{"AWS_ENDPOINT_URL": s.URL, "AWS_REGION": "test"},
			Stdin:   `{"Credentials":{"AccessKeyId":"FOO","SecretAccessKey":"BAR","Expiration":"2015-09-17T14:09:41Z"}}`,
			Stdout:  Banner + "\n",
		},

		test.ExecRun{
			Command: "convox install --stack-name Invalid",
			Exit:    1,
			Env:     map[string]string{"AWS_ENDPOINT_URL": s.URL, "AWS_REGION": "test"},
			Stdin:   `{"Credentials":{"AccessKeyId":"FOO","SecretAccessKey":"BAR","Expiration":"2015-09-17T14:09:41Z"}}`,
			Stderr:  `ERROR: Stack name is invalid, must match [a-z0-9-]*`,
		},

		test.ExecRun{
			Command: "convox install --stack-name in_valid",
			Exit:    1,
			Env:     map[string]string{"AWS_ENDPOINT_URL": s.URL, "AWS_REGION": "test"},
			Stdin:   `{"Credentials":{"AccessKeyId":"FOO","SecretAccessKey":"BAR","Expiration":"2015-09-17T14:09:41Z"}}`,
			Stderr:  `ERROR: Stack name is invalid, must match [a-z0-9-]*`,
		},
	)
}

func TestConvoxInstallFileCredentials(t *testing.T) {

}
