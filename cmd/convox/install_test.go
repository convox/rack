package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/convox/rack/api/awsutil"
	"github.com/convox/rack/test"
	"github.com/convox/version"
	"github.com/stretchr/testify/assert"
)

func TestConvoxInstallSTDINCredentials(t *testing.T) {
	stackId := "arn:aws:cloudformation:us-east-1:123456789:stack/MyStack/aaf549a0-a413-11df-adb3-5081b3858e83"
	cycles := []awsutil.Cycle{
		{
			awsutil.Request{"GET", "/", "", "/./"},
			awsutil.Response{200, `<CreateStackResult><StackId>` + stackId + `</StackId></CreateStackResult>`},
		},
		{
			awsutil.Request{"GET", "/", "", ""},
			awsutil.Response{200, ""},
		},
	}

	handler := awsutil.NewHandler(cycles)
	s := httptest.NewServer(handler)
	defer s.Close()

	os.Setenv("AWS_ENDPOINT", s.URL)
	latest, _ := version.Latest()

	test.Runs(t,
		test.ExecRun{
			Command: "convox install",
			Exit:    0,
			Env:     map[string]string{"AWS_ENDPOINT_URL": s.URL, "AWS_REGION": "test"},
			Stdin:   `{"Credentials":{"AccessKeyId":"FOO","SecretAccessKey":"test","Expiration":"2015-09-17T14:09:41Z"}}`,
			Stdout:  Banner + "\nInstalling Convox (" + latest + ")...\n" + stackId + "\n",
		},
	)
}

func TestConvoxInstallValidateStackName(t *testing.T) {
	stackId := "arn:aws:cloudformation:us-east-1:123456789:stack/MyStack/aaf549a0-a413-11df-adb3-5081b3858e83"
	cycles := []awsutil.Cycle{
		{
			awsutil.Request{"GET", "/", "", "/./"},
			awsutil.Response{200, `<CreateStackResult><StackId>` + stackId + `</StackId></CreateStackResult>`},
		},
		{
			awsutil.Request{"GET", "/", "", ""},
			awsutil.Response{200, ""},
		},
	}

	handler := awsutil.NewHandler(cycles)
	s := httptest.NewServer(handler)
	defer s.Close()

	os.Setenv("AWS_ENDPOINT", s.URL)
	latest, _ := version.Latest()

	test.Runs(t,
		test.ExecRun{
			Command: "convox install --stack-name valid",
			Exit:    0,
			Env:     map[string]string{"AWS_ENDPOINT_URL": s.URL, "AWS_REGION": "test"},
			Stdin:   `{"Credentials":{"AccessKeyId":"FOO","SecretAccessKey":"test","Expiration":"2015-09-17T14:09:41Z"}}`,
			Stdout:  Banner + "\nInstalling Convox (" + latest + ")...\n" + stackId + "\n",
		},

		test.ExecRun{
			Command: "convox install --stack-name Invalid",
			Exit:    1,
			Env:     map[string]string{"AWS_ENDPOINT_URL": s.URL, "AWS_REGION": "test"},
			Stdin:   `{"Credentials":{"AccessKeyId":"FOO","SecretAccessKey":"BAR","Expiration":"2015-09-17T14:09:41Z"}}`,
			Stderr:  `ERROR: Stack name 'Invalid' is invalid, must match [a-z0-9-]*`,
		},

		test.ExecRun{
			Command: "convox install --stack-name in_valid",
			Exit:    1,
			Env:     map[string]string{"AWS_ENDPOINT_URL": s.URL, "AWS_REGION": "test"},
			Stdin:   `{"Credentials":{"AccessKeyId":"FOO","SecretAccessKey":"BAR","Expiration":"2015-09-17T14:09:41Z"}}`,
			Stderr:  `ERROR: Stack name 'in_valid' is invalid, must match [a-z0-9-]*`,
		},
	)
}

// TestConvoxInstallFileCredentialsNonexistent ensures an error is raised when a file argument is provided but the file doesn't exist
func TestConvoxInstallFileCredentialsNonexistent(t *testing.T) {
}

// TestConvoxInstallFileCredentialsInvalidFormat ensures an error is raised when a file argument is provided but the file isn't in the expected format
func TestConvoxInstallFileCredentialsInvalidFormat(t *testing.T) {
}

// TestConvoxInstallFileCredentialsInsufficientPermissions ensures an error is raised when a file argument is provided but the user doesn't have sufficient permissions to install a Rack
func TestConvoxInstallFileCredentialsInsufficientPermissions(t *testing.T) {
	cycles := []awsutil.Cycle{
		{
			awsutil.Request{"POST", "/", "", "Action=GetUser&Version=2010-05-08"},
			awsutil.Response{403, `
			<ErrorResponse xmlns="http://iam.amazonaws.com/doc/2010-05-15/">,
				<Error>
					<Code>AccessDenied</Code>
				</Error>
				<RequestId>bc91dc86-5803-11e5-a24f-85fde26a90fa</RequestId>
			</ErrorResponse>`,
			},
		},
	}

	handler := awsutil.NewHandler(cycles)
	s := httptest.NewServer(handler)
	defer s.Close()

	os.Setenv("AWS_ENDPOINT", s.URL)
	latest, _ := version.Latest()

	test.Runs(t,
		test.ExecRun{
			Command: "convox install ../../manifest/fixtures/dummy.csv",
			Exit:    1,
			Stdout:  Banner + "\nInstalling Convox (" + latest + ")...\nReading credentials from file ../../manifest/fixtures/dummy.csv\n",
			Stderr:  "ERROR: AccessDenied. See https://docs.convox.com/creating-an-iam-user\n",
		},
	)
}

func TestConvoxInstallFriendlyName(t *testing.T) {
	var formation struct {
		Resources map[string]struct {
			Type string
		}
	}

	data, err := ioutil.ReadFile("../../provider/aws/dist/rack.json")
	assert.Nil(t, err)
	assert.NotEmpty(t, data)

	err = json.Unmarshal(data, &formation)
	assert.Nil(t, err)

	types := map[string]bool{}

	for _, r := range formation.Resources {
		types[r.Type] = true
	}

	for typ := range types {
		assert.NotContains(t, FriendlyName(typ), "Unknown")
	}
}
