package main

import (
	"fmt"
	"testing"

	"github.com/convox/cli/client"
	"github.com/convox/cli/test"
)

func TestInvalidLogin(t *testing.T) {
	ts := testServer(t,
		test.Http{Method: "GET", Path: "/apps", Code: 401, Response: "unauthorized"},
	)

	defer ts.Close()

	test.Runs(t,
		test.ExecRun{
			Command: fmt.Sprintf("convox login --password foobar %s", ts.URL),
			Exit:    1,
			Stderr:  "ERROR: invalid login\n",
		},
		test.ExecRun{
			Command: "convox login --password foobar BAD",
			Exit:    1,
			Stderr:  "ERROR: invalid login\n",
		},
	)
}

func TestLogin(t *testing.T) {
	ts := testServer(t,
		test.Http{Method: "GET", Path: "/apps", Code: 200, Response: client.Apps{}},
	)

	defer ts.Close()

	test.Runs(t,
		test.ExecRun{
			Command: fmt.Sprintf("convox login --password foobar %s", ts.URL),
			Exit:    0,
			Stdout:  "Logged in successfully.\n",
		},
	)
}
