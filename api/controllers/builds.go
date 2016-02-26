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

	b, err := models.GetBuild(app, build)

	if err != nil && strings.HasPrefix(err.Error(), "no such build") {
		return httperr.Errorf(404, err.Error())
	}

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, b)
}

func BuildCreate(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	build := models.NewBuild(mux.Vars(r)["app"])
	config := r.FormValue("config")

	if build.IsRunning() {
		return httperr.TrackErrorf("BuildCreate", "build.IsRunning", 403, "Another build is currently running. Please try again later.")
	}

	err := r.ParseMultipartForm(50 * 1024 * 1024)

	if err != nil && err != http.ErrNotMultipart {
		return httperr.TrackServer("BuildCreate", "ParseMultipartForm", err)
	}

	err = build.Save()

	if err != nil {
		return httperr.TrackServer("BuildCreate", "build.Save", err)
	}

	resources, err := models.ListResources(os.Getenv("RACK"))

	if err != nil {
		return httperr.TrackServer("BuildCreate", "models.ListResources", err)
	}

	ch := make(chan error)

	source, _, err := r.FormFile("source")

	if err != nil && err != http.ErrMissingFile && err != http.ErrNotMultipart {
		return httperr.TrackServer("BuildCreate", "FormFile", err)
	}

	cache := !(r.FormValue("cache") == "false")

	if source != nil {
		err = models.S3PutFile(resources["RegistryBucket"].Id, fmt.Sprintf("builds/%s.tgz", build.Id), source, false)

		if err != nil {
			return httperr.TrackServer("BuildCreate", "models.S3PutFile", err)
		}

		helpers.TrackSuccess("BuildCreate", "build.ExecuteLocal")

		go build.ExecuteLocal(source, cache, config, ch)

		err = <-ch

		if err != nil {
			return httperr.TrackServer("BuildExecute", "build.ExecuteLocal", err)
		} else {
			return RenderJson(rw, build)
		}
	}

	if repo := r.FormValue("repo"); repo != "" {
		helpers.TrackSuccess("BuildCreate", "build.ExecuteRemote")

		go build.ExecuteRemote(repo, cache, config, ch)

		err = <-ch

		if err != nil {
			return httperr.TrackServer("BuildExecute", "build.ExecuteRemote", err)
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

		helpers.TrackSuccess("BuildCreate", "build.ExecuteIndex")

		go build.ExecuteIndex(index, cache, config, ch)

		err = <-ch

		if err != nil {
			return httperr.TrackServer("BuildExecute", "build.ExecuteIndex", err)
		} else {
			return RenderJson(rw, build)
		}
	}

	return httperr.Errorf(403, "no source or repo")
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
