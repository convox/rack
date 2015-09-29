package main

import (
	"testing"

	"github.com/convox/rack/client"
	"github.com/convox/rack/test"
)

func TestDeployPreventAgainstCreating(t *testing.T) {
	ts := testServer(t,
		test.Http{Method: "GET", Path: "/apps/foo", Code: 200, Response: client.App{Name: "foo", Status: "creating"}},
	)

	defer ts.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox deploy --app foo",
			Exit:    1,
			Stdout:  "Deploying foo\n",
			Stderr:  "ERROR: app is still creating: foo\n",
		},
	)
}
