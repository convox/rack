package client

import (
	"net/url"
	"testing"

	"github.com/convox/cli/test"
	"github.com/stretchr/testify/assert"
)

func TestGetSystem(t *testing.T) {
	ts := testServer(t,
		test.Http{Method: "GET", Path: "/system", Code: 200, Response: System{
			Count:   1,
			Name:    "system",
			Status:  "running",
			Type:    "type",
			Version: "version",
		}},
	)

	defer ts.Close()

	system, err := testClient(t, ts.URL).GetSystem()

	assert.NotNil(t, system, "system should not be nil")
	assert.Nil(t, err, "err should be nil")

	assert.Equal(t, 1, system.Count, ".Count should be 1")
	assert.Equal(t, "system", system.Name, ".Name should be system")
	assert.Equal(t, "running", system.Status, ".Status should be running")
	assert.Equal(t, "type", system.Type, ".Type should be type")
	assert.Equal(t, "version", system.Version, ".Version should be version")
}

func TestGetSystemFailure(t *testing.T) {
	ts := testServer(t,
		test.Http{Method: "GET", Path: "/system", Code: 503, Response: Error{
			Error: "invalid system",
		}},
	)

	defer ts.Close()

	u, _ := url.Parse(ts.URL)

	client := New(u.Host, "test", "test")

	system, err := client.GetSystem()

	assert.Nil(t, system)
	assert.NotNil(t, err)
	assert.Equal(t, "invalid system", err.Error(), "err should be invalid system")
}
