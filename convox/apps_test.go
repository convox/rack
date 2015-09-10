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

	testRuns(t, ts,
		Run{
			Command: []string{"convox", "apps"},
			Stdout:  "APP      STATUS \nsinatra  running\n",
		},
	)
}

func TestAppsCreate(t *testing.T) {
	ts := httpStub(
		Stub{Method: "POST", Path: "/apps", Code: 200, Response: client.App{}},
	)

	defer ts.Close()

	testRuns(t, ts,
		Run{
			Command: []string{"convox", "apps", "create", "foobar"},
			Stdout:  "Creating app foobar... CREATING\n",
		},
	)
}

func TestAppsCreateFail(t *testing.T) {
	ts := httpStub(
		Stub{Method: "POST", Path: "/apps", Code: 403, Response: client.Error{Error: "app already exists"}},
	)

	defer ts.Close()

	testRuns(t, ts,
		Run{
			Command: []string{"convox", "apps", "create", "foobar"},
			Stdout:  "Creating app foobar... ",
			Stderr:  "ERROR: app already exists\n",
		},
	)
}
