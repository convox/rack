package controllers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/ddollar/logger"
	docker "github.com/convox/rack/Godeps/_workspace/src/github.com/fsouza/go-dockerclient"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/convox/rack/Godeps/_workspace/src/golang.org/x/net/websocket"
	"github.com/convox/rack/api/helpers"
	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/api/models"
	"github.com/convox/rack/api/provider"
)

func BuildList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	app := mux.Vars(r)["app"]

	builds, err := models.ListBuilds(app)

	if err != nil {
		return httperr.Server(err)
	}

	_, err = models.GetApp(app)

	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", app)
	}

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, builds)
}

func BuildGet(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)
	app := vars["app"]
	build := vars["build"]

	_, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", app)
	}

	b, err := provider.BuildGet(app, build)

	if err != nil && strings.HasPrefix(err.Error(), "no such build") {
		return httperr.Errorf(404, err.Error())
	}

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, b)
}

func BuildDelete(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)
	app := vars["app"]
	build := vars["build"]

	b, err := provider.BuildDelete(app, build)
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, b)
}

func BuildCreate(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)

	source, _, err := r.FormFile("source")
	if err != nil && err != http.ErrMissingFile && err != http.ErrNotMultipart {
		helpers.TrackError("build", err, map[string]interface{}{"at": "FormFile"})
		return httperr.Server(err)
	}

	if source != nil {
		// Log into private registries that we might pull from
		err := models.LoginPrivateRegistries()
		if err != nil {
			return httperr.Server(err)
		}

		app, err := models.GetApp(vars["app"])
		if err != nil {
			return httperr.Server(err)
		}

		// Log into registry that we will push to
		_, err = models.AppDockerLogin(*app)
		if err != nil {
			return httperr.Server(err)
		}

		cache := !(r.FormValue("cache") == "false")
		b, err := provider.BuildCreateTar(vars["app"], source, r.FormValue("manifest"), r.FormValue("description"), cache)
		if err != nil {
			return httperr.Server(err)
		}

		return RenderJson(rw, b)
	}

	build := models.NewBuild(mux.Vars(r)["app"])
	build.Description = r.FormValue("description")

	manifest := r.FormValue("manifest") // empty value will default "docker-compose.yml" in cmd/build

	// use deprecated "config" param if set and "manifest" is not
	if config := r.FormValue("config"); config != "" && manifest == "" {
		manifest = config
	}

	if build.IsRunning() {
		err := fmt.Errorf("Another build is currently running. Please try again later.")
		helpers.TrackError("build", err, map[string]interface{}{"at": "build.IsRunning"})
		return httperr.Server(err)
	}

	err = build.Save()

	if err != nil {
		helpers.TrackError("build", err, map[string]interface{}{"at": "build.Save"})
		return httperr.Server(err)
	}

	ch := make(chan error)

	cache := !(r.FormValue("cache") == "false")

	if repo := r.FormValue("repo"); repo != "" {
		go build.ExecuteRemote(repo, cache, manifest, ch)

		err = <-ch

		if err != nil {
			helpers.TrackError("build", err, map[string]interface{}{"at": "build.ExecuteRemote"})
			return httperr.Server(err)
		} else {
			return RenderJson(rw, build)
		}
	}

	if data := r.FormValue("index"); data != "" {
		var index models.Index

		err := json.Unmarshal([]byte(data), &index)

		if err != nil {
			return httperr.Server(err)
		}

		go build.ExecuteIndex(index, cache, manifest, ch)

		err = <-ch

		if err != nil {
			helpers.TrackError("build", err, map[string]interface{}{"at": "build.ExecuteIndex"})
			return httperr.Server(err)
		} else {
			return RenderJson(rw, build)
		}
	}

	return httperr.Errorf(403, "no source or repo")
}

func BuildUpdate(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)
	app := vars["app"]
	build := vars["build"]

	a, err := provider.AppGet(app)
	if err != nil {
		return httperr.Server(err)
	}

	b, err := provider.BuildGet(app, build)
	if err != nil {
		return httperr.Server(err)
	}

	b.Ended = time.Now()
	b.Manifest = r.FormValue("manifest")
	b.Reason = r.FormValue("reason")
	b.Status = r.FormValue("status")

	// if build was successful create a release
	if b.Status == "complete" && b.Manifest != "" {
		_, err := provider.BuildRelease(b)
		if err != nil {
			return httperr.Server(err)
		}
	}

	return RenderJson(rw, b)
}

func BuildCopy(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)
	app := vars["app"]
	build := vars["build"]

	dest := r.FormValue("app")

	_, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such source app: %s", app)
	}

	srcBuild, err := models.GetBuild(app, build)

	if err != nil && strings.HasPrefix(err.Error(), "no such build") {
		return httperr.Errorf(404, err.Error())
	}

	if err != nil {
		return httperr.Server(err)
	}

	destApp, err := models.GetApp(dest)

	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such destination app: %s", dest)
	}

	destBuild, err := srcBuild.CopyTo(*destApp)

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, destBuild)
}

func BuildLogs(ws *websocket.Conn) *httperr.Error {
	vars := mux.Vars(ws.Request())
	app := vars["app"]
	build := vars["build"]

	_, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", app)
	}

	_, err = models.GetBuild(app, build)

	if err != nil {
		return httperr.Server(err)
	}

	// default to local docker socket
	host := "unix:///var/run/docker.sock"

	// in production loop through docker hosts that the rack is running on
	// to find the build
	if os.Getenv("DEVELOPMENT") != "true" {
		pss, err := models.ListProcesses(os.Getenv("RACK"))

		if err != nil {
			return httperr.Server(err)
		}

		for _, ps := range pss {
			client, err := ps.Docker()

			if err != nil {
				return httperr.Server(err)
			}

			res, err := client.ListContainers(docker.ListContainersOptions{
				All: true,
				Filters: map[string][]string{
					"name": []string{fmt.Sprintf("build-%s", build)},
				},
			})

			if len(res) > 0 {
				host = fmt.Sprintf("http://%s:2376", ps.Host)
				break
			}
		}
	}

	fmt.Printf("host %+v\n", host)

	// proxy to docker container logs
	// https://docs.docker.com/reference/api/docker_remote_api_v1.19/#get-container-logs
	client, err := docker.NewClient(host)

	if err != nil {
		return httperr.Server(err)
	}

	r, w := io.Pipe()

	quit := make(chan bool)

	go scanLines(r, ws)
	go keepAlive(ws, quit)

	err = client.Logs(docker.LogsOptions{
		Container:    fmt.Sprintf("build-%s", build),
		Follow:       true,
		Stdout:       true,
		Stderr:       true,
		Tail:         "all",
		RawTerminal:  false,
		OutputStream: w,
		ErrorStream:  w,
	})

	quit <- true

	return httperr.Server(err)
}

func scanLines(r io.Reader, ws *websocket.Conn) {
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), "|", 2)

		if len(parts) < 2 {
			ws.Write([]byte(parts[0] + "\n"))
			continue
		}

		switch parts[0] {
		case "manifest":
		case "error":
			ws.Write([]byte(parts[1] + "\n"))
		default:
			ws.Write([]byte(parts[1] + "\n"))
		}
	}
}

func keepAlive(ws *websocket.Conn, quit chan bool) {
	c := time.Tick(5 * time.Second)
	b := []byte{}

	for {
		select {
		case <-c:
			ws.Write(b)
		case <-quit:
			return
		}
	}
}

func logEvent(log *logger.Logger, build models.Build, step string, err error) {
	if err != nil {
		log.Log("state=error step=build.%s app=%q build=%q error=%q", step, build.App, build.Id, err)
	} else {
		log.Success("step=build.%s app=%q build=%q", step, build.App, build.Id)
	}
}
