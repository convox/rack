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

	setLoginEnv(ts)

	stdout, stderr := appRun([]string{"convox", "apps"})

	expect(t, stdout, `sinatra
`)
	expect(t, stderr, "")
}

func TestAppsCreate(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/apps":
			_ = App{Name: r.FormValue("name")}
			http.Error(w, "ok", 302)

		case "/apps/foobar":
			app := App{Name: "foobar"}
			data, _ := json.Marshal(app)
			_, _ = w.Write(data)
		}
	}))
	defer ts.Close()

	setLoginEnv(ts)

	stdout, stderr := appRun([]string{"convox", "apps", "create", "--name", "foobar"})

	expect(t, stdout, "Created foobar.\n")
	expect(t, stderr, "")
}

func TestAppsCreateFail(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "app already exists", 400)
	}))
	defer ts.Close()

	setLoginEnv(ts)

	stdout, stderr := appRun([]string{"convox", "apps", "create", "--name", "foobar"})

	expect(t, stdout, "")
	expect(t, stderr, "ERROR: app already exists\n")
}
