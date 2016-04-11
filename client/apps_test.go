package client

import (
	"testing"

	"github.com/convox/rack/test"
	"github.com/stretchr/testify/assert"
)

func TestGetApps(t *testing.T) {
	ts := testServer(t,
		test.Http{Method: "GET", Path: "/apps", Code: 200, Response: Apps{
			App{Name: "sinatra", Status: "running"},
		}},
	)

	defer ts.Close()

	apps, err := testClient(t, ts.URL).GetApps()

	assert.NotNil(t, apps, "apps should not be nil")
	assert.Nil(t, err, "err should be nil")

	assert.Equal(t, 1, len(apps), 2, "there should be one app")
	assert.Equal(t, "sinatra", apps[0].Name, "app name should be sinatra")
	assert.Equal(t, "running", apps[0].Status, "app status should be running")
}

func TestGetAppsFailure(t *testing.T) {
	ts := testServer(t,
		test.Http{Method: "GET", Path: "/apps", Code: 503, Response: Error{
			Error: "error message here",
		}},
	)

	defer ts.Close()

	apps, err := testClient(t, ts.URL).GetApps()

	assert.Nil(t, apps, "apps should be nil")
	assert.NotNil(t, err, "err should not be nil")

	assert.Equal(t, "error message here", err.Error(), "error message here")
}

func TestGetApp(t *testing.T) {
	ts := testServer(t,
		test.Http{Method: "GET", Path: "/apps/sinatra", Code: 200, Response: App{
			Name:   "sinatra",
			Status: "running",
		}},
	)

	defer ts.Close()

	app, err := testClient(t, ts.URL).GetApp("sinatra")

	assert.NotNil(t, app, "apps should not be nil")
	assert.Nil(t, err, "err should be nil")

	assert.Equal(t, "sinatra", app.Name, "app name should be sinatra")
	assert.Equal(t, "running", app.Status, "app status should be running")
}

func TestGetAppNotFound(t *testing.T) {
	ts := testServer(t,
		test.Http{Method: "GET", Path: "/apps/notfound", Code: 403, Response: Error{
			Error: "not found",
		}},
	)

	defer ts.Close()

	app, err := testClient(t, ts.URL).GetApp("notfound")

	assert.Nil(t, app, "app should be nil")
	assert.NotNil(t, err, "err should not be nil")

	assert.Equal(t, "not found", err.Error(), "err should be 'not found'")
}
