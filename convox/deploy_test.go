package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/convox/cli/convox/build"
)

func TestBuildTagPush(t *testing.T) {
	m, _ := build.ManifestFromBytes([]byte(`web:
  build: .
  command: ruby web.rb
  ports:
  - 5000:3000
worker:
  build: .
  command: ruby worker.rb
redis:
  image: convox/redis
`))

	expect(t,
		m.ImageNames("myproj"),
		[]string{"convox/redis", "myproj_web", "myproj_worker"},
	)

	expect(t,
		m.TagNames("private.registry.com:5000", "myproj", "123"),
		[]string{"private.registry.com:5000/convox/redis:123", "private.registry.com:5000/myproj_web:123", "private.registry.com:5000/myproj_worker:123"},
	)
}

func TestDeploy(t *testing.T) {
	statuses := []string{"running", "running"}

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/apps":
			app := App{
				Name: r.FormValue("name"),
			}

			data, _ := json.Marshal(app)
			_, _ = w.Write(data)

		case "/apps/dockercompose":
			http.Error(w, "not found", 404)

		case "/apps/dockercompose/status":
			s := statuses[0]
			statuses = append(statuses[:0], statuses[1:]...)
			_, _ = w.Write([]byte(s))

		case "/apps/dockercompose/releases":
			_, _ = w.Write([]byte("ok"))
		}
	}))
	defer ts.Close()

	setLoginEnv(ts)

	base, _ := filepath.Abs(".")
	project := filepath.Join(base, "..", "examples", "docker-compose")

	stdout, stderr := appRun([]string{"convox", "deploy", project})

	expect(t, stdout, `Docker Compose app detected.
tag httpd 127.0.0.1:5000/httpd:1435444444
Created app dockercompose
Status running
Created release 1435444444
Status running
`)
	expect(t, stderr, "")
}
