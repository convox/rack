package client

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/convox/cli/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testClient(t *testing.T, serverUrl string) *Client {
	u, _ := url.Parse(serverUrl)

	client, err := New(u.Host, "test", "test")

	require.NotNil(t, client, "client should not be nil")
	require.Nil(t, err, "err should be nil")

	return client
}

func testServer(t *testing.T, stubs ...test.Http) *httptest.Server {
	stubs = append(stubs, test.Http{Method: "GET", Path: "/system", Code: 200, Response: System{
		Version: "test",
	}})

	return test.Server(t, stubs...)
}

type ErrorReader struct {
	Error string
}

func (er ErrorReader) Read(buf []byte) (int, error) {
	return 0, fmt.Errorf(er.Error)
}

func (er ErrorReader) Close() error {
	return nil
}

func TestClientRackNoVersion(t *testing.T) {
	ts := testServer(t,
		test.Http{Method: "GET", Path: "/system", Code: 200, Response: System{
			Count:   1,
			Name:    "system",
			Status:  "running",
			Type:    "type",
			Version: "",
		}},
	)

	u, _ := url.Parse(ts.URL)

	client, err := New(u.Host, "test", "test")

	assert.Nil(t, client, "client is nil")
	assert.NotNil(t, err, "err is not nil")
	assert.Equal(t, "rack outdated, please update with `convox rack update`", err.Error())
}

func TestClientRackOldVersion(t *testing.T) {
	ts := testServer(t,
		test.Http{Method: "GET", Path: "/system", Code: 200, Response: System{
			Count:   1,
			Name:    "system",
			Status:  "running",
			Type:    "type",
			Version: "1",
		}},
	)

	u, _ := url.Parse(ts.URL)

	MinimumServerVersion = "2"

	client, err := New(u.Host, "test", "test")

	assert.Nil(t, client, "client is nil")
	assert.NotNil(t, err, "err is not nil")
	assert.Equal(t, "rack outdated, please update with `convox rack update`", err.Error())
}

func TestClientErrorReading(t *testing.T) {
	er := ErrorReader{Error: "error reading"}
	res := &http.Response{StatusCode: 400, Body: er}
	err := responseError(res)

	assert.NotNil(t, err, "err is not nil")
	assert.Equal(t, "error reading response body: error reading", err.Error(), "err text is valid")
}

func TestClientNonJson(t *testing.T) {
	ts := testServer(t,
		test.Http{Method: "GET", Path: "/", Code: 503, Response: "not-json"},
	)

	defer ts.Close()

	var err Error

	testClient(t, ts.URL).Get("/", &err)

	fmt.Printf("err: %+v\n", err)
}
