package main

import (
	"testing"

	"github.com/convox/rack/client"
	"github.com/convox/rack/test"
)

// TestEnvGetAll ensures the environment of an app can be read.
func TestEnvGetAll(t *testing.T) {
	ts := testServer(t,
		test.Http{
			Method: "GET",
			Path:   "/apps/myapp/environment",
			Code:   200,
			Response: client.Environment{
				"foo": "bar",
				"baz": "qux",
			},
		},
	)

	defer ts.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox env -a myapp",
			Exit:    0,
			Stdout:  "baz=qux\nfoo=bar\n",
		},
	)
}

// TestEnvGet ensures a single environment variable can be read
func TestEnvGet(t *testing.T) {
	ts := testServer(t,
		test.Http{
			Method: "GET",
			Path:   "/apps/myapp/environment",
			Code:   200,
			Response: client.Environment{
				"foo": "bar",
				"baz": "qux",
			},
		},
	)

	defer ts.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox env get -a myapp baz",
			Exit:    0,
			Stdout:  "qux\n",
		},
	)
}

// TestGetEnvNoVariableSpecified ensures an error is raised when `convox env get` is run without arguments.
func TestGetEnvNoVariableSpecified(t *testing.T) {
	ts := testServer(t,
		test.Http{
			Method:   "GET",
			Path:     "/apps/myapp/environment",
			Code:     200,
			Response: client.Environment{},
		},
	)

	defer ts.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox env get",
			Exit:    1,
			Stderr:  "No variable specified",
		},
	)
}

// TestEnvGetNoSuchEnvVar ensures an empty string is returned when a user reads a variable that doesn't exist.
func TestEnvGetNoSuchEnvVar(t *testing.T) {
	ts := testServer(t,
		test.Http{
			Method:   "GET",
			Path:     "/apps/myapp/environment",
			Code:     200,
			Response: client.Environment{},
		},
	)

	defer ts.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox env get -a myapp baz",
			Exit:    0,
			Stdout:  "\n",
		},
	)
}

// TestEnvGetNoSuchApp ensures an error is raised when a user requests the environment for an app that doesn't exist.
func TestEnvGetNoSuchApp(t *testing.T) {
	ts := testServer(t,
		test.Http{
			Method:   "GET",
			Path:     "/apps/myapp/environment",
			Code:     404,
			Response: client.Error{Error: "no such app"},
		},
	)

	defer ts.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox env get -a myapp baz",
			Exit:    1,
			Stderr:  "ERROR: no such app\n",
		},
	)
}

/* TestEnvSet tests setting an environment variable.
Note: 'env set' first retrieves the app's current environment,
then appends each argument to it and sets the result as the new environment.
*/
func TestEnvSet(t *testing.T) {
	ts := testServer(t,
		test.Http{
			Method: "POST",
			Path:   "/apps/myapp/environment",
			Body:   "foo=bar\n",
			Code:   200,
		},
		test.Http{
			Method:   "GET",
			Path:     "/apps/myapp/environment",
			Code:     200,
			Response: client.Environment{},
		},
	)

	defer ts.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox env set -a myapp foo=bar",
			Exit:    0,
			Stdout:  "Updating environment... OK\n",
		},
	)
}

// TestEnvSetStdin ensures environment variables can be set by piping a file to `convox env set`.
func TestEnvSetStdin(t *testing.T) {
	ts := testServer(t,
		test.Http{
			Method: "POST",
			Path:   "/apps/myapp/environment",
			Body:   "foo=bar\nping=pong\n",
			Code:   200,
		},
		test.Http{
			Method:   "GET",
			Path:     "/apps/myapp/environment",
			Code:     200,
			Response: client.Environment{},
		},
	)

	defer ts.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "cat ../../manifest/fixtures/env-test.env | convox env set -a myapp",
			Exit:    0,
			Stdout:  "Updating environment... OK\n",
		},
	)
}

// TestEnvSetNoValue ensures an error is raised when a user attempts to set a variable with no value.
func TestEnvSetNoValue(t *testing.T) {
	ts := testServer(t,
		test.Http{
			Method: "POST",
			Path:   "/apps/myapp/environment",
			Body:   "foo=\n",
			Code:   200,
		},
		test.Http{
			Method:   "GET",
			Path:     "/apps/myapp/environment",
			Code:     200,
			Response: client.Environment{},
		},
	)

	defer ts.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox env set -a myapp foo=",
			Exit:    1,
			Stderr:  "ERROR: Can't set foo to an empty value; try `convox env unset`.",
		},
	)
}

// TestEnvApi ensures an app's environment can be read via `convox api get`.
func TestEnvApi(t *testing.T) {
	ts := testServer(t,
		test.Http{
			Method:   "GET",
			Path:     "/apps/myapp/environment",
			Code:     200,
			Response: client.Environment{},
		},
	)

	defer ts.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox api get /apps/myapp/environment",
			Exit:    0,
			Stdout:  "{}\n",
		},
	)
}
