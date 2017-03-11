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

func TestReleasesCmd(t *testing.T) {
	ts := testServer(t,
		test.Http{Method: "GET", Path: "/apps/foo", Code: 200, Response: client.App{Name: "foo", Status: "running"}},
		test.Http{Method: "GET", Path: "/apps/foo/releases", Code: 200, Response: client.Releases{client.Release{}}},
	)
	defer ts.Close()

	tests := []test.ExecRun{
		test.ExecRun{
			Command: "convox releases --app foo",
			Stdout:  "ID  CREATED  BUILD  STATUS\n                    active\n",
		},
		test.ExecRun{
			Command: "convox releases -a foo",
			Stdout:  "ID  CREATED  BUILD  STATUS\n                    active\n",
		},
		test.ExecRun{
			Command: "convox --app foo releases",
			Stdout:  "ID  CREATED  BUILD  STATUS\n                    active\n",
		},
		test.ExecRun{
			Command: "convox -a foo releases",
			Stdout:  "ID  CREATED  BUILD  STATUS\n                    active\n",
		},
	}

	for _, myTest := range tests {
		test.Runs(t, myTest)
	}
}
