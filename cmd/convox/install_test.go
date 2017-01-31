package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
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

	data, err := ioutil.ReadFile("../../provider/aws/dist/rack.json")
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

/* TestUrls checks that each URL returns HTTP status code 200.
These URLs are printed in user-facing messages and have been gathered manually.
Sources (mostly): cmd/convox/doctor.go, cmd/convox/install.go
See also: bin/check-links
*/
func TestUrls(t *testing.T) {
	urls := []string{
		iamUserURL,
		"https://convox.com/docs/about-resources/",
		"https://convox.com/docs/api-keys/",
		"https://convox.com/docs/docker-compose-file/",
		"https://convox.com/docs/troubleshooting/",
		"https://convox.com/guide/balancers/",
		"https://convox.com/guide/databases/",
		"https://convox.com/guide/environment/",
		"https://convox.com/guide/images",
		"https://convox.com/guide/links/",
		"https://convox.com/guide/one-off-commands/",
		"https://convox.com/guide/services/",
		"https://docs.docker.com/engine/installation/",
		"https://docs.docker.com/engine/reference/builder/",
		"https://git-scm.com/docs/gitignore",
		"https://github.com/convox/release",
		"https://guides.github.com/introduction/flow/",
		"http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/template-custom-resources.html",
	}

	for _, url := range urls {
		resp, err := http.Get(url)
		assert.NoError(t, err)
		rc := resp.StatusCode
		if rc != 200 {
			assert.Fail(t, fmt.Sprintf("Got response code %d for URL %s", rc, url))
		}
	}
}
