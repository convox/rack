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
		test.Http{Method: "POST", Path: "/apps/foo/builds", Body: "cache=true&config=&description=&url=https%3A%2F%2Fexample.org", Code: 200, Response: client.Build{}},
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

func TestBuildsCreateInvalidUrl(t *testing.T) {
	// TODO: Re-enable when we upgrade to Go 1.8
	return

	ts := testServer(t,
		test.Http{Method: "GET", Path: "/apps/site-git", Code: 200, Response: client.App{Name: "site-git", Status: "running"}},
	)

	defer ts.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox build -a site-git git@github.com:convox/site.git",
			Exit:    1,
			Stdout:  "",
			Stderr:  "ERROR: parse git@github.com:convox/site.git: first path segment in URL cannot contain colon\n",
		},
	)
}
