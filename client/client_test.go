package client

import (
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/convox/cli/test"
	"github.com/stretchr/testify/assert"
)

func testClient(t *testing.T, serverUrl string) *Client {
	u, _ := url.Parse(serverUrl)

	client, err := New(u.Host, "test", "test")

	assert.NotNil(t, client, "client should not be nil")
	assert.Nil(t, err, "err should be nil")

	return client
}

func testServer(t *testing.T, stubs ...test.Http) *httptest.Server {
	stubs = append(stubs, test.Http{Method: "GET", Path: "/system", Code: 200, Response: System{
		Version: "test",
	}})

	return test.Server(t, stubs...)
}
