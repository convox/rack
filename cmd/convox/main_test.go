package main

import (
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/convox/rack/client"
	"github.com/convox/rack/test"
	"github.com/stretchr/testify/assert"
)

var configlessEnv = map[string]string{
	// reset HOME to a location where there's not likely to be a convox config on the host
	"HOME":                  "/tmp/probablyNoConfigFileHere",
	"AWS_SECRET_ACCESS_KEY": "",
	"AWS_ACCESS_KEY_ID":     "",
	"CONVOX_HOST":           "",
}
var DebuglessEnv = map[string]string{"CONVOX_DEBUG": ""}
var DebugfulEnv = map[string]string{"CONVOX_DEBUG": "true"}

func testServer(t *testing.T, stubs ...test.Http) *httptest.Server {
	stubs = append(stubs, test.Http{Method: "GET", Path: "/system", Code: 200, Response: client.System{
		Version: "latest",
	}})

	server := test.Server(t, stubs...)

	u, _ := url.Parse(server.URL)

	os.Setenv("CONVOX_HOST", u.Host)
	os.Setenv("CONVOX_PASSWORD", "test")

	return server
}

func TestVersion(t *testing.T) {
	// Ensure we don't segfault if user is not logged in
	test.Runs(t, test.ExecRun{
		Command: "convox -v",
		Env:     configlessEnv,
		Exit:    1,
		Stdout:  "client: dev\n",
		Stderr:  "ERROR: no host config found, try `convox login`\n",
	})
	v := Version
	assert.Equal(t, v, "dev")
}
