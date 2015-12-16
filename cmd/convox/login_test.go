package main

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/convox/rack/client"
	"github.com/convox/rack/test"
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
			Stderr:  "ERROR: invalid login\n",
		},
		test.ExecRun{
			Command: "convox login --password foobar BAD",
			Env:     map[string]string{"CONVOX_CONFIG": temp},
			Exit:    1,
			Stderr:  "ERROR",
		},
	)
}

func TestLogin(t *testing.T) {
	temp, _ := ioutil.TempDir("", "convox-test")

	ts := testServer(t,
		test.Http{Method: "GET", Path: "/apps", Code: 200, Response: client.Apps{}},
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
