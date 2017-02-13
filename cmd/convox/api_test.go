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
