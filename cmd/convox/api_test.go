package main

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/convox/rack/client"
	"github.com/convox/rack/cmd/convox/stdcli"
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

var apiHelp = `convox api: api endpoint

Usage:
  convox api <command> [args...]

Subcommands: (convox api help <subcommand>)
  get		get an api endpoint
  delete	delete an api endpoint
  help, h

Options:
  --rack value	rack name
  --help, -h	show help

`

var apiGetHelp = `convox api get: get an api endpoint

Usage:
  convox api get <endpoint>

Options:
   --rack value	rack name

`

var commandStrings = []string{
	"convox %s api",
	"convox api %s",
}

var commandGetStrings = []string{
	"convox %s api get",
	"convox %s api get /foo",
	"convox api %s get",
	"convox api %s get /foo",
	"convox api get %s",
	"convox api get %s /foo",
}

// TODO: These commands don't behave as expected
var skipCommands = []string{
	"convox api --help",          // treats '--help' as an argument
	"convox api get help",        // treats 'help' as an argument
	"convox api get help /foo",   // treats 'help' as an argument
	"convox api get h",           // treats 'h' as an argument
	"convox api get h /foo",      // treats 'h' as an argument
	"convox api --help get",      // executes command, ignores --help
	"convox api --help get /foo", // executes command, ignores --help
	"convox api -h",              // executes command, ignores -h
	"convox api -h get",          // executes command, ignores -h
	"convox api -h get /foo",     // executes command, ignores -h
	"convox help api",            // /!\ different output from 'convox api --help'
	"convox help api get",        // outputs 'convox api' help
	"convox help api get /foo",   // outputs 'convox api' help
	"convox --help api get /foo", // Outputs 'convox' help
	"convox -h api get /foo",     // outputs 'convox' help
	"convox -h api get",          // outputs 'convox' help
	"convox -h api",              // outputs 'convox' help
	"convox h api",               // /!\ different output from 'convox api --help'
	"convox h api get",           // outputs 'convox api' help
	"convox h api get /foo",      // outputs 'convox api' help
	"convox --help api get",      // outputs 'convox' help
	"convox --help api",          // outputs 'convox' help
}

func shouldSkip(c string, skipCommands []string) bool {
	// skip permutations that don't work as expected yet
	for _, skipC := range skipCommands {
		if c == skipC {
			return true
		}
	}
	return false
}

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

	// base 'api' command (without subcommands)
	// these commands should output a help screen about 'convox api'
	for _, cmd := range commandStrings {
		for _, hf := range stdcli.HelpFlags {
			c := fmt.Sprintf(cmd, hf)

			if shouldSkip(c, skipCommands) {
				fmt.Println("SKIPPED: ", c)
				continue
			}
			test.Runs(t,
				test.ExecRun{
					Command: c,
					Exit:    0,
					Stdout:  apiHelp,
				},
			)
		}
	}

	assert.Equal(t, stdcli.HelpFlags, []string{"--help", "-h", "h", "help"})

	// 'api get' subcommand
	// these commands should output a help screen about 'convox api get'
	for _, cmd := range commandGetStrings {
		for _, hf := range stdcli.HelpFlags {
			c := fmt.Sprintf(cmd, hf)

			if shouldSkip(c, skipCommands) {
				fmt.Println("SKIPPED: ", c)
				continue
			}

			test.Runs(t,
				test.ExecRun{
					Command: c,
					Exit:    0,
					Stdout:  apiGetHelp,
				},
			)
		}
	}
}

func TestRackFlag(t *testing.T) {
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

	test.Runs(t,
		test.ExecRun{
			Command:  "convox --rack myorg/staging api get /foo",
			Exit:     0,
			OutMatch: "bar",
		},
	)

	test.Runs(t,
		test.ExecRun{
			Command:  "convox api --rack myorg/staging get /foo",
			Exit:     0,
			OutMatch: "bar",
		},
	)

	test.Runs(t,
		test.ExecRun{
			Command:  "convox api get --rack myorg/staging /foo",
			Exit:     0,
			OutMatch: "bar",
		},
	)

	test.Runs(t,
		test.ExecRun{
			Command:  "convox api get /foo --rack myorg/staging",
			Exit:     0,
			OutMatch: "bar",
		},
	)
}

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
			Stdout:  "[\n  {\n    \"name\": \"sinatra\",\n    \"release\": \"\",\n    \"status\": \"running\"\n  }\n]\n",
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
			Command: "convox api get",
			Exit:    129,
			Stdout:  apiGetHelp,
		},
	)
}
