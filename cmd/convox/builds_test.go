package main

import (
	"testing"

	"github.com/convox/rack/client"
	"github.com/convox/rack/test"
)

func TestBuildsPreventAgainstCreating(t *testing.T) {
	ts := testServer(t,
		test.Http{Method: "GET", Path: "/apps/foo", Code: 200, Response: client.App{Name: "foo", Status: "creating"}},
	)

	defer ts.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox build https://example.org --app foo",
			Exit:    1,
			Stdout:  "",
			Stderr:  "ERROR: app foo is still being created, for more information try `convox apps info`\n",
		},
	)
}

func TestBuildsCreateReturnsNoBuild(t *testing.T) {
	ts := testServer(t,
		test.Http{Method: "GET", Path: "/apps/foo", Code: 200, Response: client.App{Name: "foo", Status: "running"}},
		test.Http{Method: "POST", Path: "/apps/foo/builds", Body: "cache=true&config=docker-compose.yml&description=&url=https%3A%2F%2Fexample.org", Code: 200, Response: client.Build{}},
	)

	defer ts.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox build https://example.org --app foo",
			Exit:    1,
			Stdout:  "",
			Stderr:  "ERROR: unable to fetch build id\n",
		},
	)
}
