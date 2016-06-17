package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/convox/rack/api/helpers"
	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/api/models"
	"github.com/convox/rack/api/provider"
	"github.com/convox/rack/api/structs"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/gorilla/mux"
	"golang.org/x/net/websocket"
)

func BuildList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	app := mux.Vars(r)["app"]

	l := r.URL.Query().Get("limit")

	var err error
	var limit int

	if l == "" {
		limit = 20
	} else {
		limit, err = strconv.Atoi(l)
		if err != nil {
			return httperr.Errorf(400, err.Error())
		}
	}

	builds, err := provider.BuildList(app, int64(limit))
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

	b, err := provider.BuildGet(app, build)
	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", app)
	}
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
	app := vars["app"]

	cache := !(r.FormValue("cache") == "false")
	manifest := r.FormValue("manifest")
	description := r.FormValue("description")

	repo := r.FormValue("repo")
	index := r.FormValue("index")

	source, _, err := r.FormFile("source")
	if err != nil && err != http.ErrMissingFile && err != http.ErrNotMultipart {
		helpers.TrackError("build", err, map[string]interface{}{"at": "FormFile"})
		return httperr.Server(err)
	}

	// Log into private registries that we might pull from
	// TODO: move to prodiver BuildCreate
	err = models.LoginPrivateRegistries()
	if err != nil {
		return httperr.Server(err)
	}

	a, err := models.GetApp(app)
	if err != nil {
		return httperr.Server(err)
	}

	// Log into registry that we will push to
	_, err = models.AppDockerLogin(*a)
	if err != nil {
		return httperr.Server(err)
	}

	var b *structs.Build

	// if source file was posted, build from tar
	if source != nil {
		b, err = provider.BuildCreateTar(app, source, r.FormValue("manifest"), r.FormValue("description"), cache)
	} else if repo != "" {
		b, err = provider.BuildCreateRepo(app, repo, r.FormValue("manifest"), r.FormValue("description"), cache)
	} else if index != "" {
		var i structs.Index
		err := json.Unmarshal([]byte(index), &i)
		if err != nil {
			return httperr.Server(err)
		}

		b, err = provider.BuildCreateIndex(app, i, manifest, description, cache)
	} else {
		return httperr.Errorf(403, "no source, repo or index")
	}

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, b)
}

func BuildUpdate(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)
	app := vars["app"]
	build := vars["build"]
	didComplete := false

	b, err := provider.BuildGet(app, build)
	if err != nil {
		return httperr.Server(err)
	}

	if d := r.FormValue("description"); d != "" {
		b.Description = d
	}

	if m := r.FormValue("manifest"); m != "" {
		b.Manifest = m
	}

	if r := r.FormValue("reason"); r != "" {
		b.Reason = r
	}

	if s := r.FormValue("status"); s != "" {
		if b.Status != s && s == "complete" {
			didComplete = true
		}
		b.Status = s
		b.Ended = time.Now()
	}

	// if build was successful create a release
	if b.Status == "complete" && b.Manifest != "" {
		_, err := provider.BuildRelease(b)
		if err != nil {
			return httperr.Server(err)
		}
	}

	err = provider.BuildSave(b)
	if err != nil {
		return httperr.Server(err)
	}

	// AWS currently has a limit of 500 images in ECR
	// This is a "hopefully temporary" and brute force means
	// of preventing hitting limit errors during deployment
	if didComplete {
		bs, err := provider.BuildList(app, 150)
		if err != nil {
			fmt.Println("Error listing builds for cleanup")
		} else {
			if len(bs) >= 50 {
				wg := new(sync.WaitGroup)
				outDated := bs[50:]
				for _, b := range outDated {
					wg.Add(1)
					go func(buildId string, wg *sync.WaitGroup) {
						defer wg.Done()
						_, err := provider.BuildDelete(app, buildId)
						if err != nil {
							fmt.Printf("Error cleaning up build: %s", buildId)
						}
					}(b.Id, wg)
				}
				wg.Wait()
			}
		}
	}

	if b.Status == "failed" {
		provider.EventSend(&structs.Event{
			Action: "build:create",
			Data: map[string]string{
				"app": b.App,
				"id":  b.Id,
			},
		}, fmt.Errorf(b.Reason))
	}

	return RenderJson(rw, b)
}

func BuildCopy(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)
	srcApp := vars["app"]
	build := vars["build"]
	dest := r.FormValue("app")

	b, err := provider.BuildCopy(srcApp, build, dest)
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, b)
}

func BuildLogs(ws *websocket.Conn) *httperr.Error {
	vars := mux.Vars(ws.Request())

	app := vars["app"]
	build := vars["build"]

	_, err := provider.BuildGet(app, build)
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

	// proxy to docker container logs
	// https://docs.docker.com/reference/api/docker_remote_api_v1.19/#get-container-logs
	client, err := docker.NewClient(host)

	if err != nil {
		return httperr.Server(err)
	}

	quit := make(chan bool)

	go keepAlive(ws, quit)

	err = client.Logs(docker.LogsOptions{
		Container:    fmt.Sprintf("build-%s", build),
		Follow:       true,
		Stdout:       true,
		Stderr:       true,
		Tail:         "all",
		RawTerminal:  false,
		OutputStream: ws,
		ErrorStream:  ws,
	})

	quit <- true

	return httperr.Server(err)
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
