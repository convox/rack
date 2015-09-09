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

	stdout, stderr := appRun([]string{"convox", "login", "--password", "foobar", ts.URL})

	expect(t, stdout, "")
	expect(t, stderr, "ERROR: invalid login\n")
}

func TestLogin(t *testing.T) {
	ts := httpStub(
		Stub{Method: "GET", Path: "/apps", Code: 200, Response: client.Apps{}},
	)

	defer ts.Close()

	stdout, stderr := appRun([]string{"convox", "login", "--password", "foobar", ts.URL})

	expect(t, stdout, "Logged in successfully.\n")
	expect(t, stderr, "")
}

func TestLoginHost(t *testing.T) {
	ts := httpStub(
		Stub{Method: "GET", Path: "/apps", Code: 200, Response: client.Apps{}},
	)

	defer ts.Close()

	stdout, stderr := appRun([]string{"convox", "login", "--password", "foobar", ts.URL})

	expect(t, stdout, "Logged in successfully.\n")
	expect(t, stderr, "")

	stdout, stderr = appRun([]string{"convox", "login", "--password", "foobar", "BAD"})

	expect(t, stdout, "")
	expect(t, stderr, "ERROR: Get https://BAD/apps: dial tcp: lookup BAD: no such host\n")
}
