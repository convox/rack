package main

import (
	"testing"

	"github.com/convox/cli/client"
)

func TestApps(t *testing.T) {
	ts := httpStub(
		Stub{Method: "GET", Path: "/apps", Code: 200, Response: client.Apps{
			client.App{Name: "sinatra", Status: "running"},
		}},
	)

	defer ts.Close()

	stdout, stderr := appRun([]string{"convox", "apps"})

	expect(t, stdout, "APP      STATUS \nsinatra  running\n")
	expect(t, stderr, "")
}

func TestAppsCreate(t *testing.T) {
	ts := httpStub(
		Stub{Method: "POST", Path: "/apps", Code: 200, Response: client.App{}},
	)

	defer ts.Close()

	stdout, stderr := appRun([]string{"convox", "apps", "create", "foobar"})

	expect(t, stdout, "Creating app foobar... CREATING\n")
	expect(t, stderr, "")
}

func TestAppsCreateFail(t *testing.T) {
	ts := httpStub(
		Stub{Method: "POST", Path: "/apps", Code: 403, Response: client.Error{Error: "app already exists"}},
	)

	defer ts.Close()

	stdout, stderr := appRun([]string{"convox", "apps", "create", "foobar"})

	expect(t, stdout, "Creating app foobar... ")
	expect(t, stderr, "ERROR: app already exists\n")
}
