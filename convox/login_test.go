package main

import (
	"testing"

	"github.com/convox/cli/client"
)

func TestInvalidLogin(t *testing.T) {
	ts := httpStub(
		Stub{Method: "GET", Path: "/apps", Code: 401, Response: "unauthorized"},
	)

	defer ts.Close()

	testRuns(t,
		Run{
			Command: []string{"convox", "login", "--password", "foobar", ts.URL},
			Stderr:  "ERROR: invalid login\n",
		},
	)
}

func TestLogin(t *testing.T) {
	ts := httpStub(
		Stub{Method: "GET", Path: "/apps", Code: 200, Response: client.Apps{}},
	)

	defer ts.Close()

	testRuns(t,
		Run{
			Command: []string{"convox", "login", "--password", "foobar", ts.URL},
			Stdout:  "Logged in successfully.\n",
		},
		Run{
			Command: []string{"convox", "login", "--password", "foobar", "BAD"},
			Stderr:  "ERROR: Get https://BAD/system: dial tcp: lookup BAD: no such host\n",
		},
	)
}
