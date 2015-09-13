package main

import (
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/convox/cli/client"
	"github.com/convox/cli/test"
)

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
