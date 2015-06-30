package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/convox/cli/convox/build"
)

func TestBuildTagPush(t *testing.T) {
	m, _ := build.ManifestFromBytes([]byte(`web:
  image: httpd
  ports:
  - 80:80
worker:
  build: .
  command: ruby worker.rb
redis:
  image: convox/redis
`))

	tags := m.Tags("private.registry.com:5000", "myproj", "123")
	expect(t, tags, map[string]string{
		"private.registry.com:5000/myproj-redis:123":  "convox/redis",
		"private.registry.com:5000/myproj-web:123":    "httpd",
		"private.registry.com:5000/myproj-worker:123": "myproj_worker",
	})
}

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
					"Release": "1435444444",
				},
			}
			data, _ := json.Marshal(app)
			_, _ = w.Write(data)

		case "/apps/docker-compose/status":
			s := statuses[0]
			statuses = append(statuses[:0], statuses[1:]...)
			_, _ = w.Write([]byte(s))

		case "/apps/docker-compose/releases":
			_, _ = w.Write([]byte("ok"))
		default:
			http.Error(w, fmt.Sprintf("Not Found: %s", r.URL.Path), 500)
		}
	}))
	defer ts.Close()

	setLoginEnv(ts)

	base, _ := filepath.Abs(".")
	project := filepath.Join(base, "..", "examples", "docker-compose")

	stdout, stderr := appRun([]string{"convox", "deploy", project})

	expect(t, stdout, `Docker Compose app detected.
Tagging httpd
Pushing 127.0.0.1:5000/dockercompose-web:1435444444
Releasing 1435444444
Name         docker-compose
Status       running
Release      1435444444
`)
	expect(t, stderr, "")
}
