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
			Name:   "sinatra",
			Status: "running",
		})

		data, _ := json.Marshal(apps)
		_, _ = w.Write(data)
	}))
	defer ts.Close()

	setLoginEnv(ts)

	stdout, stderr := appRun([]string{"convox", "apps"})

	expect(t, stdout, "APP      STATUS\nsinatra  running\n")
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

		case "/apps/foobar/status":
			w.Write([]byte("running"))
		}
	}))
	defer ts.Close()

	setLoginEnv(ts)

	stdout, stderr := appRun([]string{"convox", "apps", "create", "foobar"})

	expect(t, stdout, "Creating app foobar: OK\n")
	expect(t, stderr, "")
}

func TestAppsCreateFail(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "app already exists", 400)
	}))
	defer ts.Close()

	setLoginEnv(ts)

	stdout, stderr := appRun([]string{"convox", "apps", "create", "foobar"})

	expect(t, stdout, "Creating app foobar: ")
	expect(t, stderr, "ERROR: app already exists\n")
}
