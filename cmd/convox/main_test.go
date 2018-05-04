package main

import (
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/convox/rack/client"
	"github.com/convox/rack/test"
)

var configlessEnv = map[string]string{
	// reset HOME to a location where there's not likely to be a convox config on the host
	"HOME":        "/tmp/probablyNoConfigFileHere",
	"CONVOX_HOST": "",
}
var DebuglessEnv = map[string]string{"CONVOX_DEBUG": ""}
var DebugfulEnv = map[string]string{"CONVOX_DEBUG": "true"}

func testServer(t *testing.T, stubs ...test.Http) *httptest.Server {
	stubs = append(stubs, test.Http{Method: "GET", Path: "/system", Code: 200, Response: client.System{
		Version: "latest",
	}})

	stubs = append(stubs, test.Http{Method: "GET", Path: "/racks", Code: 200, Response: []client.Rack{
		client.Rack{
			Name: "test",
		},
	}})

	server := test.Server(t, stubs...)

	u, _ := url.Parse(server.URL)

	os.Setenv("CONVOX_HOST", u.Host)
	os.Setenv("CONVOX_PASSWORD", "test")

	return server
}
