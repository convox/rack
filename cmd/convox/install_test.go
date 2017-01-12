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
			Stdout:  Banner + "\nInstalling Convox (" + latest + ")...\nUsing AWS Access Key ID: FOO\n" + stackId + "\n",
		},
	)
}

// TestConvoxInstallEnvCredentials ensures credentials are read from the environment when present
func TestConvoxInstallEnvCredentials(t *testing.T) {
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
			Env: map[string]string{
				"AWS_ENDPOINT_URL":      s.URL,
				"AWS_REGION":            "test",
				"AWS_ACCESS_KEY_ID":     "test",
				"AWS_SECRET_ACCESS_KEY": "test",
			},
			Stdout: Banner + "\nInstalling Convox (" + latest + ")...\nUsing AWS Access Key ID: test\n" + stackId + "\n",
		},
	)
}

// TestConvoxInstallFileCredentials ensures credentials are read from a file when one is provided
func TestConvoxInstallFileCredentials(t *testing.T) {
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
			Command: "convox install ../../manifest/fixtures/dummy.csv",
			Exit:    0,
			Env: map[string]string{
				"AWS_ENDPOINT_URL": s.URL,
				"AWS_REGION":       "test",
			},
			Stdout: Banner + "\nInstalling Convox (" + latest + ")...\nReading credentials from file ../../manifest/fixtures/dummy.csv\nUsing AWS Access Key ID: AKIAIJAFAQL3V7HLQQAA\n" + stackId + "\n",
		},
	)
}

// TestConvoxInstallFileCredentialsWithEnvCredentials ensures credentials are read from a file when one is provided, even if environmental credentials are present
func TestConvoxInstallFileCredentialsWithEnvCredentials(t *testing.T) {
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
			Command: "convox install ../../manifest/fixtures/dummy.csv",
			Exit:    0,
			Env: map[string]string{
				"AWS_ENDPOINT_URL":      s.URL,
				"AWS_REGION":            "test",
				"AWS_ACCESS_KEY_ID":     "test",
				"AWS_SECRET_ACCESS_KEY": "test",
			},
			Stdout: Banner + "\nInstalling Convox (" + latest + ")...\nReading credentials from file ../../manifest/fixtures/dummy.csv\nUsing AWS Access Key ID: AKIAIJAFAQL3V7HLQQAA\n" + stackId + "\n",
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
			Stdout:  Banner + "\nInstalling Convox (" + latest + ")...\nUsing AWS Access Key ID: FOO\n" + stackId + "\n",
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
	latest, _ := version.Latest()

	test.Runs(t,
		test.ExecRun{
			Command: "convox install ../../manifest/fixtures/nothinghere.csv",
			Exit:    1,
			Stdout:  Banner + "\nInstalling Convox (" + latest + ")...\nReading credentials from file ../../manifest/fixtures/nothinghere.csv\n",
			Stderr:  "ERROR: open ../../manifest/fixtures/nothinghere.csv: no such file or directory\n",
		},
	)
}

// TestConvoxInstallFileCredentialsInvalidFormat ensures an error is raised when a file argument is provided but the file isn't in the expected format
func TestConvoxInstallFileCredentialsInvalidFormat(t *testing.T) {
	latest, _ := version.Latest()

	test.Runs(t,
		test.ExecRun{
			Command: "convox install ../../manifest/fixtures/invalid-credentials-format.csv",
			Exit:    1,
			Stdout:  Banner + "\nInstalling Convox (" + latest + ")...\nReading credentials from file ../../manifest/fixtures/invalid-credentials-format.csv\n",
			Stderr:  "ERROR: Credentials file ../../manifest/fixtures/invalid-credentials-format.csv is not in a valid format; line 1 should contain headers: \nUser name Password Access key ID Secret access key Console login link\nInstead it contains: \nUser Name Access Key Id Secret Access Key\n",
		},
	)
}

// TestConvoxInstallFileCredentialsInsufficientPermissions ensures an error is raised when a file argument is provided but the user doesn't have sufficient permissions to install a Rack
func TestConvoxInstallFileCredentialsInsufficientPermissions(t *testing.T) {
	// TODO: disabled along with validateUserAccess() in cmd/convox/install.go
	return
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
