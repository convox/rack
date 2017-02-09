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
