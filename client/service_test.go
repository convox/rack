package client

import (
	"net/url"
	"testing"

	"github.com/convox/rack/test"
	"github.com/stretchr/testify/assert"
)

func TestGetService(t *testing.T) {
	ts := testServer(t,
		test.Http{Method: "GET", Path: "/resources/convox-events", Code: 200, Response: Resource{
			Name:   "convox-events",
			Status: "running",
			Type:   "type",
		},
		},
	)

	defer ts.Close()

	service, err := testClient(t, ts.URL).GetResource("convox-events")

	assert.NotNil(t, service, "service should not be nil")
	assert.NoError(t, err)

	assert.Equal(t, "convox-events", service.Name, ".Name should be convox-events")
	assert.Equal(t, "running", service.Status, ".Status should be running")
	assert.Equal(t, "type", service.Type, ".Type should be type")
}

func TestGetServiceFailure(t *testing.T) {
	ts := testServer(t,
		test.Http{Method: "GET", Path: "/resources/nonexistent", Code: 503, Response: Error{
			Error: "invalid resource",
		}},
	)

	defer ts.Close()

	u, _ := url.Parse(ts.URL)

	client := New(u.Host, "test", "test")

	service, err := client.GetResource("nonexistent")

	assert.Nil(t, service)
	assert.NotNil(t, err)
	assert.Equal(t, "invalid resource", err.Error(), "err should be invalid resource")
}
