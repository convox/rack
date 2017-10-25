package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/convox/rack/api/awsutil"
	"github.com/convox/rack/test"
	"github.com/convox/version"
	"github.com/stretchr/testify/assert"
)

func TestConvoxInstallSTDINCredentials(t *testing.T) {
	stackID := "arn:aws:cloudformation:us-east-1:123456789:stack/MyStack/aaf549a0-a413-11df-adb3-5081b3858e83"
	cycles := []awsutil.Cycle{
		{
			awsutil.Request{"GET", "/", "", "/./"},
			awsutil.Response{200, `<CreateStackResult><StackId>` + stackID + `</StackId></CreateStackResult>`},
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
			Stdout:  Banner + "\nInstalling Convox (" + latest + ")...\nUsing AWS Access Key ID: FOO\n" + stackID + "\n",
		},
	)
}

// TestConvoxInstallEnvCredentials ensures credentials are read from the environment when present
func TestConvoxInstallEnvCredentials(t *testing.T) {
	stackID := "arn:aws:cloudformation:us-east-1:123456789:stack/MyStack/aaf549a0-a413-11df-adb3-5081b3858e83"
	cycles := []awsutil.Cycle{
		{
			awsutil.Request{"GET", "/", "", "/./"},
			awsutil.Response{200, `<CreateStackResult><StackId>` + stackID + `</StackId></CreateStackResult>`},
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
			Stdout: Banner + "\nInstalling Convox (" + latest + ")...\nUsing AWS Access Key ID: test\n" + stackID + "\n",
		},
	)
}

// TestConvoxInstallFileCredentials ensures credentials are read from a file when one is provided
func TestConvoxInstallFileCredentials(t *testing.T) {
	stackID := "arn:aws:cloudformation:us-east-1:123456789:stack/MyStack/aaf549a0-a413-11df-adb3-5081b3858e83"
	cycles := []awsutil.Cycle{
		{
			awsutil.Request{"GET", "/", "", "/./"},
			awsutil.Response{200, `<CreateStackResult><StackId>` + stackID + `</StackId></CreateStackResult>`},
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
			Command: "convox install ./data/fixtures/dummy.csv",
			Exit:    0,
			Env: map[string]string{
				"AWS_ENDPOINT_URL": s.URL,
				"AWS_REGION":       "test",
			},
			Stdout: Banner + "\nInstalling Convox (" + latest + ")...\nReading credentials from file ./data/fixtures/dummy.csv\nUsing AWS Access Key ID: AKIAIJAFAQL3V7HLQQAA\n" + stackID + "\n",
		},
	)
}

// TestConvoxInstallFileCredentialsWithEnvCredentials ensures credentials are read from a file when one is provided, even if environmental credentials are present
func TestConvoxInstallFileCredentialsWithEnvCredentials(t *testing.T) {
	stackID := "arn:aws:cloudformation:us-east-1:123456789:stack/MyStack/aaf549a0-a413-11df-adb3-5081b3858e83"
	cycles := []awsutil.Cycle{
		{
			awsutil.Request{"GET", "/", "", "/./"},
			awsutil.Response{200, `<CreateStackResult><StackId>` + stackID + `</StackId></CreateStackResult>`},
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
			Command: "convox install ./data/fixtures/dummy.csv",
			Exit:    0,
			Env: map[string]string{
				"AWS_ENDPOINT_URL":      s.URL,
				"AWS_REGION":            "test",
				"AWS_ACCESS_KEY_ID":     "test",
				"AWS_SECRET_ACCESS_KEY": "test",
			},
			Stdout: Banner + "\nInstalling Convox (" + latest + ")...\nReading credentials from file ./data/fixtures/dummy.csv\nUsing AWS Access Key ID: AKIAIJAFAQL3V7HLQQAA\n" + stackID + "\n",
		},
	)
}

func TestConvoxInstallValidateStackName(t *testing.T) {
	stackID := "arn:aws:cloudformation:us-east-1:123456789:stack/MyStack/aaf549a0-a413-11df-adb3-5081b3858e83"
	cycles := []awsutil.Cycle{
		{
			awsutil.Request{"GET", "/", "", "/./"},
			awsutil.Response{200, `<CreateStackResult><StackId>` + stackID + `</StackId></CreateStackResult>`},
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
			Stdout:  Banner + "\nInstalling Convox (" + latest + ")...\nUsing AWS Access Key ID: FOO\n" + stackID + "\n",
		},

		test.ExecRun{
			Command: "convox install --stack-name Invalid",
			Exit:    1,
			Env:     map[string]string{"AWS_ENDPOINT_URL": s.URL, "AWS_REGION": "test"},
			Stdin:   `{"Credentials":{"AccessKeyId":"FOO","SecretAccessKey":"BAR","Expiration":"2015-09-17T14:09:41Z"}}`,
			Stderr:  `ERROR: stack name 'Invalid' is invalid, must match [a-z0-9-]*`,
		},

		test.ExecRun{
			Command: "convox install --stack-name in_valid",
			Exit:    1,
			Env:     map[string]string{"AWS_ENDPOINT_URL": s.URL, "AWS_REGION": "test"},
			Stdin:   `{"Credentials":{"AccessKeyId":"FOO","SecretAccessKey":"BAR","Expiration":"2015-09-17T14:09:41Z"}}`,
			Stderr:  `ERROR: stack name 'in_valid' is invalid, must match [a-z0-9-]*`,
		},
	)
}

// TestConvoxInstallFileCredentialsNonexistent ensures an error is raised when a file argument is provided but the file doesn't exist
func TestConvoxInstallFileCredentialsNonexistent(t *testing.T) {
	latest, _ := version.Latest()

	test.Runs(t,
		test.ExecRun{
			Command: "convox install ./data/fixtures/nothinghere.csv",
			Exit:    1,
			Stdout:  Banner + "\nInstalling Convox (" + latest + ")...\nReading credentials from file ./data/fixtures/nothinghere.csv\n",
			Stderr:  "ERROR: open ./data/fixtures/nothinghere.csv: no such file or directory\n",
		},
	)
}

// TestConvoxInstallFileCredentialsInvalidFormat ensures an error is raised when a file argument is provided but the file isn't in the expected format
func TestConvoxInstallFileCredentialsInvalidFormat(t *testing.T) {
	latest, _ := version.Latest()

	test.Runs(t,
		test.ExecRun{
			Command: "convox install ./data/fixtures/credswrong.csv",
			Exit:    1,
			Stdout:  Banner + "\nInstalling Convox (" + latest + ")...\nReading credentials from file ./data/fixtures/credswrong.csv\n",
			Stderr:  "ERROR: credentials secrets is of unknown length\n",
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

	data, err := ioutil.ReadFile("../../provider/aws/formation/rack.json")
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	err = json.Unmarshal(data, &formation)
	assert.NoError(t, err)

	types := map[string]bool{}

	for _, r := range formation.Resources {
		types[r.Type] = true
	}

	for typ := range types {
		assert.NotContains(t, FriendlyName(typ), "Unknown")
	}
}

func TestReadCredentialsFromFile(t *testing.T) {

	creds, err := readCredentialsFromFile("./data/fixtures/creds2.csv")
	assert.Nil(t, err)
	assert.Equal(t, "fakeaccessid", creds.Access)
	assert.Equal(t, "fakesecretkey", creds.Secret)

	creds, err = readCredentialsFromFile("./data/fixtures/creds3.csv")
	assert.Nil(t, err)
	assert.Equal(t, "fakeaccessid3", creds.Access)
	assert.Equal(t, "fakesecretkey3", creds.Secret)

	creds, err = readCredentialsFromFile("./data/fixtures/creds5.csv")
	assert.Nil(t, err)
	assert.Equal(t, "fakeaccessid", creds.Access)
	assert.Equal(t, "fakesecretkey", creds.Secret)

	creds, err = readCredentialsFromFile("./data/fixtures/credswrong.csv")
	assert.EqualError(t, err, "credentials secrets is of unknown length")

	creds, err = readCredentialsFromFile("./data/fixtures/credswrong2.csv")
	assert.EqualError(t, err, "credentials file is of unknown length")
}

func TestRequiredFlagsWhenInstallingIntoExistingVPC(t *testing.T) {
	test.Runs(t,
		test.ExecRun{
			Command: "convox install --existing-vpc foo",
			Exit:    1,
			Stdout:  "WARNING: [existing vpc] using default subnet cidrs (10.0.1.0/24,10.0.2.0/24,10.0.3.0/24); if this is incorrect, pass a custom value to --subnet-cidrs\nWARNING: [existing vpc] using default vpc cidr (10.0.0.0/16); if this is incorrect, pass a custom value to --vpc-cidr\n",
			Stderr:  "ERROR: must specify --internet-gateway for existing VPC\n",
		},
	)
}

/* TestUrls checks that each URL returns HTTP status code 200.
These URLs are printed in user-facing messages and have been gathered manually.
Sources (mostly): cmd/convox/doctor.go, cmd/convox/install.go
See also: bin/check-links
*/
func TestUrls(t *testing.T) {
	urls := []string{
		"https://convox.com/docs/about-resources/",
		"https://convox.com/docs/api-keys/",
		"https://convox.com/docs/docker-compose-file/",
		"https://convox.com/docs/dockerfile/",
		"https://convox.com/docs/environment",
		"https://convox.com/docs/one-off-commands/",
		"https://convox.com/docs/troubleshooting/",
		"https://docs.docker.com/engine/installation/",
		"https://docs.docker.com/engine/reference/builder/",
		"https://git-scm.com/docs/gitignore",
		"https://github.com/convox/release",
		"https://guides.github.com/introduction/flow/",
		"http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/template-custom-resources.html",
	}

	tr := &http.Transport{
		TLSHandshakeTimeout: 5 * time.Second,
	}

	client := &http.Client{Transport: tr}

	for _, url := range urls {
		resp, err := client.Get(url)
		assert.NoError(t, err)
		rc := resp.StatusCode
		if rc != 200 {
			assert.Fail(t, fmt.Sprintf("Got response code %d for URL %s", rc, url))
		}
	}
}

func TestInstallCmd(t *testing.T) {
	tests := []test.ExecRun{
		// help flags
		test.ExecRun{
			Command:  "convox install -h",
			OutMatch: "convox install: install convox into an aws account",
		},

		// no credentials
		// FIXME: test suite doesn't handle standard input properly (Stdin behaves as if the input were piped to the command)
		test.ExecRun{
			Env:      configlessEnv,
			Command:  "convox install",
			OutMatch: "This installer needs AWS credentials to install/uninstall the Convox platform",
			Stderr:   "ERROR: EOF\n",
			Exit:     1,
		},
	}
	key := os.Getenv("AWS_ACCESS_KEY_ID")
	secret := os.Getenv("AWS_SECRET_ACCESS_KEY")

	os.Unsetenv("AWS_ACCESS_KEY_ID")
	os.Unsetenv("AWS_SECRET_ACCESS_KEY")

	for _, myTest := range tests {
		test.Runs(t, myTest)
	}

	if key != "" && secret != "" {
		os.Setenv("AWS_ACCESS_KEY_ID", key)
		os.Setenv("AWS_SECRET_ACCESS_KEY", secret)
	}
}

func TestAwsCLICredentialsNil(t *testing.T) {
	home := os.Getenv("HOME")

	os.Setenv("HOME", configlessEnv["HOME"])

	creds, err := awsCLICredentials()
	assert.Nil(t, creds)
	assert.NoError(t, err)

	os.Setenv("HOME", home)
}
