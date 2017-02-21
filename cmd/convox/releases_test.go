package main

import (
	"testing"

	"github.com/convox/rack/client"
	"github.com/convox/rack/test"
)

func TestPromotePreventAgainstCreating(t *testing.T) {
	ts := testServer(t,
		test.Http{Method: "GET", Path: "/apps/foo", Code: 200, Response: client.App{Name: "foo", Status: "creating"}},
	)

	defer ts.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox releases promote xxxxxxxx --app foo",
			Exit:    1,
			Stdout:  "",
			Stderr:  "ERROR: app foo is still being updated, check `convox apps info`\n",
		},
	)
}
