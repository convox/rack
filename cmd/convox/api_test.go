package main

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/convox/rack/client"
	"github.com/convox/rack/test"
	"github.com/stretchr/testify/assert"
)

func TestApiGet(t *testing.T) {
	ts := testServer(t,
		test.Http{Method: "GET", Path: "/foo", Code: 200, Response: "bar"},
	)
	defer ts.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox api get /foo",
			Exit:    0,
			Stdout:  "\"bar\"\n",
		},
	)
}

/* HELP CHECKS */
// http://www.gnu.org/prep/standards/html_node/_002d_002dhelp.html

var apiUsages = []string{
	`convox api: make a rest api call to a convox endpoint`,
	`convox api <command> <endpoint> [options]`,
	`Subcommands: (convox api <subcommand> --help)
  get      make a GET request to an api endpoint
  delete   make a DELETE request to an api endpoint
  help, h`,
	`Options:
  --rack value  rack name
  --help, -h    show help`}

var apiGetUsages = []string{
	`convox api get: make a GET request to an api endpoint`,
	`convox api get <endpoint> [options]`,
	`Options:
  --rack value  rack name`,
}

var endpointsUsage = `
Valid endpoints:
  /apps
  /apps/<app-name>
  /auth
  /certificates
  /index
  /instances
  /racks
  /registries
  /resources
  /switch
  /system`

var apiMissingEndpoint = `ERROR: 1 argument is required: <endpoint>
`
var apiMissingSubcommand = `ERROR: Missing expected subcommand
`

func TestApiHelpFlag(t *testing.T) {
	ts := testServer(t,
		test.Http{
			Method:   "GET",
			Path:     "/foo",
			Code:     200,
			Response: "bar",
			Headers:  map[string]string{"Rack": "myorg/staging"},
		},
	)
	defer ts.Close()

	tests := []test.ExecRun{
		test.ExecRun{
			Command:    "convox api",
			OutMatches: apiUsages,
			Stderr:     apiMissingSubcommand,
		},
		test.ExecRun{
			Command:    "convox api h",
			OutMatches: apiUsages,
		},
		test.ExecRun{
			Command:    "convox api help",
			OutMatches: apiUsages,
		},
		test.ExecRun{
			Command:    "convox api -h",
			OutMatches: apiUsages,
		},
		test.ExecRun{
			Command:    "convox api --help",
			OutMatches: apiUsages,
		},

		// api get
		test.ExecRun{
			Command:    "convox api get",
			OutMatches: apiGetUsages,
			Stderr:     apiMissingEndpoint,
			Exit:       129,
		},
		test.ExecRun{
			Command:    "convox api get h",
			OutMatches: apiGetUsages,
		},
		test.ExecRun{
			Command:    "convox api get help",
			OutMatches: apiGetUsages,
		},
		test.ExecRun{
			Command:    "convox api get -h",
			OutMatches: apiGetUsages,
		},
		test.ExecRun{
			Command:    "convox api get --help",
			OutMatches: apiGetUsages,
		},
		test.ExecRun{
			Command:    "convox api h get",
			OutMatches: apiGetUsages,
		},
		test.ExecRun{
			Command:    "convox api help get",
			OutMatches: apiGetUsages,
		},

		// undesired behavior
		test.ExecRun{
			Command:    "convox api -h get",
			OutMatches: apiUsages,
		},

		// too many args
		test.ExecRun{
			Command:    "convox api get foo bar",
			Env:        DebuglessEnv,
			OutMatches: apiGetUsages,
			Exit:       129,
		},
		test.ExecRun{
			Command:    "convox api get foo bar",
			Env:        DebugfulEnv,
			Stderr:     "ERROR: expected 1 argument <endpoint>; got 2 arguments (foo bar).\n",
			OutMatches: apiGetUsages,
			Exit:       129,
		},
	}

	for _, myTest := range tests {
		test.Runs(t, myTest)
	}
}

// func TestRackFlag(t *testing.T) {
//   ts := testServer(t,
//     test.Http{
//       Method:   "GET",
//       Path:     "/foo",
//       Code:     200,
//       Response: "bar",
//       Headers:  map[string]string{"Rack": "myorg/staging"},
//     },
//   )
//   defer ts.Close()

//   test.Runs(t,
//     test.ExecRun{
//       Command:  "convox --rack myorg/staging api get /foo",
//       Exit:     0,
//       OutMatch: "bar",
//     },
//   )

//   test.Runs(t,
//     test.ExecRun{
//       Command:  "convox api --rack myorg/staging get /foo",
//       Exit:     0,
//       OutMatch: "bar",
//     },
//   )

//   test.Runs(t,
//     test.ExecRun{
//       Command:  "convox api get --rack myorg/staging /foo",
//       Exit:     0,
//       OutMatch: "bar",
//     },
//   )

//   test.Runs(t,
//     test.ExecRun{
//       Command:  "convox api get /foo --rack myorg/staging",
//       Exit:     0,
//       OutMatch: "bar",
//     },
//   )
// }

// TestApiGetRequest should ensure Content-Type header is set to application/json during 'convox api get'
func TestApiGetRequest(t *testing.T) {
	host, err := currentHost()
	assert.NoError(t, err)
	c := client.New(host, "testPassword", "testVersion")
	req, err := c.Request("GET", "/nonexistent", nil)
	assert.NoError(t, err)
	b64pw := base64.StdEncoding.EncodeToString([]byte("convox:testPassword"))
	assert.Equal(t, req.Header.Get("Authorization"), fmt.Sprintf("Basic %s", b64pw))
	assert.Equal(t, req.Header.Get("Content-Type"), "application/json")
	assert.Equal(t, req.Header.Get("Version"), "testVersion")
}
func TestApiGetApps(t *testing.T) {
	ts := testServer(t,
		test.Http{Method: "GET", Path: "/apps", Code: 200, Response: client.Apps{
			client.App{Name: "sinatra", Status: "running"},
		}},
	)

	defer ts.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox api get /apps",
			Exit:    0,
			Stdout:  "[\n  {\n    \"generation\": \"\",\n    \"name\": \"sinatra\",\n    \"release\": \"\",\n    \"sleep\": false,\n    \"status\": \"running\"\n  }\n]\n",
		},
	)
}

// TestApiGet404 should ensure an error is returned when a user runs 'convox api get' with an invalid API endpoint.
func TestApiGet404(t *testing.T) {
	ts := testServer(t,
		test.Http{
			Method:   "GET",
			Path:     "/nonexistent",
			Code:     404,
			Response: client.Error{Error: "A wild 404 appears!"},
		},
	)
	defer ts.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox api get /nonexistent",
			Exit:    1,
			Stderr:  "ERROR: A wild 404 appears!",
		},
	)
}

// TestApiGetNoArg should ensure help text is displayed when user runs 'api get' without an endpoint.
func TestApiGetNoArg(t *testing.T) {
	ts := testServer(t,
		test.Http{
			Method: "GET",
			Code:   129,
		},
	)
	defer ts.Close()

	test.Runs(t,
		test.ExecRun{
			Command:    "convox api get",
			Exit:       129,
			OutMatches: apiGetUsages,
		},
	)
}

// TestApiTrailingSlash should ensure we fall back to /endpoint when user runs `convox api get /endpoint/`
func TestApiTrailingSlash(t *testing.T) {
	ts := testServer(t,
		test.Http{Method: "GET", Path: "/apps", Code: 200, Response: client.Apps{
			client.App{Name: "sinatra", Status: "running"},
		}},
	)

	defer ts.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox api get /apps/",
			Exit:    0,
			Stdout:  "[\n  {\n    \"generation\": \"\",\n    \"name\": \"sinatra\",\n    \"release\": \"\",\n    \"sleep\": false,\n    \"status\": \"running\"\n  }\n]\n",
		},
	)
}
