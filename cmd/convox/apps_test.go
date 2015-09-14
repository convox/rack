package main

import (
	"testing"

	"github.com/convox/cli/client"
	"github.com/convox/cli/test"
)

func TestApps(t *testing.T) {
	ts := testServer(t,
		test.Http{Method: "GET", Path: "/apps", Code: 200, Response: client.Apps{
			client.App{Name: "sinatra", Status: "running"},
		}},
	)

	defer ts.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox apps",
			Exit:    0,
			Stdout:  "APP      STATUS \nsinatra  running\n",
		},
	)
}

func TestAppsCreate(t *testing.T) {
	ts := testServer(t,
		test.Http{Method: "POST", Path: "/apps", Code: 200, Response: client.App{}},
	)

	defer ts.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox apps create foobar",
			Exit:    0,
			Stdout:  "Creating app foobar... CREATING\n",
		},
	)
}

func TestAppsCreateFail(t *testing.T) {
	ts := testServer(t,
		test.Http{Method: "POST", Path: "/apps", Code: 403, Response: client.Error{Error: "app already exists"}},
	)

	defer ts.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox apps create foobar",
			Exit:    1,
			Stdout:  "Creating app foobar... ",
			Stderr:  "ERROR: app already exists\n",
		},
	)
}
