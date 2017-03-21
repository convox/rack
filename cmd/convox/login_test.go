package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/convox/rack/client"
	"github.com/convox/rack/test"
	"github.com/stretchr/testify/assert"
)

func TestInvalidLogin(t *testing.T) {
	temp, _ := ioutil.TempDir("", "convox-test")

	ts := testServer(t,
		test.Http{Method: "GET", Path: "/apps", Code: 401, Response: "unauthorized"},
		test.Http{Method: "GET", Path: "/auth", Code: 404, Response: "not found"},
	)

	defer ts.Close()

	test.Runs(t,
		test.ExecRun{
			Command: fmt.Sprintf("convox login --password foobar %s", ts.URL),
			Env:     map[string]string{"CONVOX_CONFIG": temp},
			Exit:    1,
			Stderr:  "ERROR: invalid login\nHave you created an account at https://convox.com/signup?\n",
		},
		test.ExecRun{
			Command: "convox login --password foobar BAD",
			Env:     map[string]string{"CONVOX_CONFIG": temp},
			Exit:    1,
			Stderr:  "ERROR",
		},
	)
}

func TestLoginConsole(t *testing.T) {
	temp, _ := ioutil.TempDir("", "convox-test")

	ts := testServer(t,
		test.Http{Method: "GET", Path: "/auth", Code: 200, Response: map[string]string{"id": "somestring"}},
	)

	defer ts.Close()

	test.Runs(t,
		test.ExecRun{
			Command: fmt.Sprintf("convox login --password foobar %s", ts.URL),
			Env:     map[string]string{"CONVOX_CONFIG": temp},
			Exit:    0,
			Stdout:  "Logged in successfully.\n",
		},
	)
}

func TestLoginRack(t *testing.T) {
	temp, _ := ioutil.TempDir("", "convox-test")

	ts := testServer(t,
		test.Http{Method: "GET", Path: "/auth", Code: 200, Response: client.Auth{}},
	)

	defer ts.Close()

	test.Runs(t,
		test.ExecRun{
			Command: fmt.Sprintf("convox login --password foobar %s", ts.URL),
			Env:     map[string]string{"CONVOX_CONFIG": temp},
			Exit:    0,
			Stdout:  "Logged in successfully.\n",
		},
	)
}

func TestLoginFuncs(t *testing.T) {
	temp, _ := ioutil.TempDir("", "convox-test")
	oldConfig := os.Getenv("CONVOX_CONFIG")
	oldHost := os.Getenv("CONVOX_HOST")
	oldPassword := os.Getenv("CONVOX_PASSWORD")

	/* test currentLogin() with dummy values */
	os.Setenv("CONVOX_CONFIG", temp)
	os.Setenv("CONVOX_HOST", "testHost")
	os.Setenv("CONVOX_PASSWORD", "testPassword")

	host, password, err := currentLogin()
	assert.NoError(t, err)
	assert.Equal(t, "testHost", host)
	assert.Equal(t, "testPassword", password)

	/* Now try with an empty config */
	// Note: Can't just export HOME here, because ConfigRoot has already been initialized
	ConfigRoot = "/tmp/nothinghere"
	assert.Equal(t, "/tmp/nothinghere", ConfigRoot)
	os.Unsetenv("CONVOX_CONFIG")
	os.Unsetenv("CONVOX_HOST")
	os.Unsetenv("CONVOX_PASSWORD")

	host, password, err = currentLogin()
	assert.EqualError(t, err, "no host config found")
	assert.Equal(t, "", host)
	assert.Equal(t, "", password)

	os.Setenv("CONVOX_CONFIG", oldConfig)
	os.Setenv("CONVOX_HOST", oldHost)
	os.Setenv("CONVOX_PASSWORD", oldPassword)
}
