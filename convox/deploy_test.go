package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestDeploy(t *testing.T) {
	statuses := []string{"running", "running"}

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/apps":
			app := App{Name: r.FormValue("name")}
			data, _ := json.Marshal(app)
			_, _ = w.Write(data)

		case "/apps/docker-compose":
			app := App{
				Name:   "docker-compose",
				Status: "running",
				Parameters: map[string]string{
					"Release": "RELEASEID",
				},
			}
			data, _ := json.Marshal(app)
			_, _ = w.Write(data)

		case "/apps/docker-compose/build":
			w.Write([]byte("RELEASEID"))

		case "/apps/docker-compose/builds/RELEASEID":
			build := Build{Status: "complete"}
			data, _ := json.Marshal(build)
			w.Write(data)

		case "/apps/docker-compose/status":
			s := statuses[0]
			statuses = append(statuses[:0], statuses[1:]...)
			_, _ = w.Write([]byte(s))

		default:
			http.Error(w, fmt.Sprintf("Not Found: %s", r.URL.Path), 500)
		}
	}))
	defer ts.Close()

	setLoginEnv(ts)

	base, _ := filepath.Abs(".")
	project := filepath.Join(base, "..", "examples", "docker-compose")

	stdout, stderr := appRun([]string{"convox", "deploy", project})

	expect(t, stdout, `Building. done
Releasing. done
Name         docker-compose
Status       running
Release      RELEASEID
`)
	expect(t, stderr, "")
}
