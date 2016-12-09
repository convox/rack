package main

import (
	"testing"

	"github.com/convox/rack/client"
	"github.com/convox/rack/test"
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
			Stdout:  "APP      STATUS\nsinatra  running\n",
		},
	)
}

func TestAppsCreate(t *testing.T) {
	ts := testServer(t,
		test.Http{Method: "POST", Path: "/apps", Body: "name=foobar", Code: 200, Response: client.App{}},
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

func TestAppsCreateWithDotsInName(t *testing.T) {
	ts := testServer(t,
		test.Http{Method: "POST", Path: "/apps", Body: "name=foo.bar", Code: 403, Response: client.Error{Error: "invalid name"}},
	)

	defer ts.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox apps create foo.bar",
			Exit:    0,
			Stdout:  "Creating app foo.bar... ERROR: app name can contain only alphanumeric characters, dashes and must be between 4 and 30 characters\n",
		},
	)
}

func TestAppsCreateFail(t *testing.T) {
	ts := testServer(t,
		test.Http{Method: "POST", Path: "/apps", Body: "name=foobar", Code: 400, Response: client.Error{Error: "app already exists"}},
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
