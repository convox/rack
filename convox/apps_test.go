package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestApps(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apps := make(Apps, 0)
		apps = append(apps, App{
			Name: "sinatra",
		})

		data, _ := json.Marshal(apps)
		_, _ = w.Write(data)
	}))
	defer ts.Close()

	_, _ = appRun([]string{"convox", "login", "--password", "foobar", ts.URL})
	stdout, stderr := appRun([]string{"convox", "apps"})

	expect(t, stdout, `sinatra
`)
	expect(t, stderr, "")
}

func TestAppsCreate(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app := App{
			Name: r.FormValue("name"),
		}

		data, _ := json.Marshal(app)
		_, _ = w.Write(data)
	}))
	defer ts.Close()

	_, _ = appRun([]string{"convox", "login", "--password", "foobar", ts.URL})
	stdout, stderr := appRun([]string{"convox", "apps", "create", "--name", "foobar"})

	expect(t, stdout, "Created foobar.\n")
	expect(t, stderr, "")
}
