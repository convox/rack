package controllers

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/convox/rack/api/helpers"
	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/api/models"
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

	builds, err := models.Provider().BuildList(app, int64(limit))
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

	b, err := models.Provider().BuildGet(app, build)
	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", app)
	}
	if err != nil && strings.HasPrefix(err.Error(), "no such build") {
		return httperr.Errorf(404, err.Error())
	}
	if err != nil {
		return httperr.Server(err)
	}

	l, err := models.Provider().BuildLogs(app, build)
	if err != nil {
		return httperr.Server(err)
	}

	b.Logs = l

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

	a, err := models.Provider().AppGet(app)
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
		b, err = models.Provider().BuildCreateTar(app, source, r.FormValue("manifest"), r.FormValue("description"), cache)
	} else if repo != "" {
		b, err = models.Provider().BuildCreateRepo(app, repo, r.FormValue("manifest"), r.FormValue("description"), cache)
	} else if index != "" {
		var i structs.Index
		err := json.Unmarshal([]byte(index), &i)
		if err != nil {
			return httperr.Server(err)
		}

		b, err = models.Provider().BuildCreateIndex(app, i, manifest, description, cache)
	} else {
		return httperr.Errorf(403, "no source, repo or index")
	}

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, b)
}

// BuildDelete deletes a build. Makes sure not to delete a build that is contained in the active release
func BuildDelete(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)
	appName := vars["app"]
	buildID := vars["build"]

	active, err := isBuildActive(appName, buildID)
	if err != nil {
		return httperr.Errorf(404, err.Error())
	}
	if active {
		return httperr.Errorf(400, "cannot delete build contained in active release")
	}

	err = models.Provider().ReleaseDelete(appName, buildID)
	if err != nil {
		return httperr.Server(err)
	}

	build, err := models.Provider().BuildDelete(appName, buildID)
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, build)
}

// BuildImport imports a build for an app
func BuildImport(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)
	app := vars["app"]

	source, _, err := r.FormFile("source")
	if err != nil && err != http.ErrMissingFile && err != http.ErrNotMultipart {
		helpers.TrackError("build.import", err, map[string]interface{}{"at": "FormFile"})
		return httperr.Server(err)
	}

	a, err := models.Provider().AppGet(app)
	if err != nil {
		return httperr.Server(err)
	}

	build, images, err := readImportArtifact(source)
	if err != nil {
		return httperr.Server(err)
	}

	// Log into registry that we will push to
	_, err = models.AppDockerLogin(*a)
	if err != nil {
		return httperr.Server(err)
	}

	// load the images to repo
	for _, img := range images {
		cmd := exec.Command("docker", "load")
		cmd.Stdin = bytes.NewReader(img)

		out, err := cmd.Output()
		output := string(out)
		if err != nil {
			return httperr.Server(fmt.Errorf("docker load failed: %s", err))
		}

		fmt.Printf("fn=BuildImport at=DockerLoad level=info msg=\"%s\"\n", output)

		loadPrefix := "Loaded image: "
		if !strings.HasPrefix(output, loadPrefix) {
			return httperr.Server(fmt.Errorf("unexpected docker load output: %s", output))
		}

		imageSplit := strings.Split(output, loadPrefix)
		if len(imageSplit) < 2 {
			return httperr.Server(fmt.Errorf("docker load output split failed: %s", output))
		}

		tag := strings.Split(imageSplit[1], ":")[1]

		repo, err := models.Provider().AppRepository(a.Name)
		if err != nil {
			return httperr.Server(err)
		}

		newName := fmt.Sprintf("%s:%s", repo.URI, strings.TrimSpace(tag))
		cmd = exec.Command("docker", "tag", strings.TrimSpace(imageSplit[1]), newName)

		out, err = cmd.Output()
		if err != nil {
			return httperr.Server(fmt.Errorf("docker tag failed: %s", err))
		}

		//TODO: Remove the orignal import tag (from imageSplit) if it didn't originally exist

		fmt.Printf("fn=BuildImport at=DockerTag level=info msg=\"new tag %s\"\n", newName)

		cmd = exec.Command("docker", "push", newName)
		out, err = cmd.Output()
		if err != nil {
			return httperr.Server(fmt.Errorf("docker push failed: %s", err))
		}
	}

	rel, err := ForkRelease(a)
	if err != nil {
		return httperr.Server(err)
	}

	rel.Build = build.Id
	rel.Manifest = build.Manifest
	err = models.Provider().ReleaseSave(rel, a.Outputs["Settings"], a.Parameters["Key"])
	if err != nil {
		return httperr.Server(err)
	}

	build.Release = rel.Id
	build.App = a.Name
	err = models.Provider().BuildSave(build)
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, build)
}

func BuildUpdate(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)
	app := vars["app"]
	build := vars["build"]
	didComplete := false

	b, err := models.Provider().BuildGet(app, build)
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
		_, err := models.Provider().BuildRelease(b)
		if err != nil {
			return httperr.Server(err)
		}
	}

	err = models.Provider().BuildSave(b)
	if err != nil {
		return httperr.Server(err)
	}

	// AWS currently has a limit of 500 images in ECR
	// This is a "hopefully temporary" and brute force means
	// of preventing hitting limit errors during deployment
	if didComplete {
		bs, err := models.Provider().BuildList(app, 150)
		if err != nil {
			fmt.Println("Error listing builds for cleanup")
		} else {
			if len(bs) >= 50 {

				go func() {
					for _, b := range bs[50:] {
						active, err := isBuildActive(app, b.Id)
						if err != nil || active {
							continue
						}

						err = models.Provider().ReleaseDelete(app, b.Id)
						if err != nil {
							fmt.Printf("Error cleaning up releases for %s: %s", b.Id, err.Error())
							continue
						}

						_, err = models.Provider().BuildDelete(app, b.Id)
						if err != nil {
							fmt.Printf("Error cleaning up build: %s", b.Id)
						}

						time.Sleep(1 * time.Second)
					}
				}()
			}
		}
	}

	if b.Status == "failed" {
		models.Provider().EventSend(&structs.Event{
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

	b, err := models.Provider().BuildCopy(srcApp, build, dest)
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, b)
}

// BuildExport creats an artifact, representing a build, to be used with another Rack
func BuildExport(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)
	app := vars["app"]
	build := vars["build"]

	b, err := models.Provider().BuildGet(app, build)
	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", app)
	}
	if err != nil && strings.HasPrefix(err.Error(), "no such build") {
		return httperr.Errorf(404, err.Error())
	}
	if err != nil {
		return httperr.Server(err)
	}

	buf := bytes.NewBuffer(make([]byte, 0))
	err = models.Provider().BuildExport(app, b.Id, buf)
	if err != nil {
		return httperr.Server(err)
	}

	rw.Header().Set("Content-Type", "application/octet-stream")
	_, err = rw.Write(buf.Bytes())

	return httperr.Server(err)
}

func BuildLogs(ws *websocket.Conn) *httperr.Error {
	vars := mux.Vars(ws.Request())

	app := vars["app"]
	build := vars["build"]

	_, err := models.Provider().BuildGet(app, build)
	if err != nil {
		return httperr.Server(err)
	}
	// default to local docker socket
	host := "unix:///var/run/docker.sock"

	// in production loop through docker hosts that the rack is running on
	// to find the build
	if os.Getenv("DEVELOPMENT") != "true" {
		h, err := findBuildHost(build)
		if err != nil {
			return httperr.Server(err)
		}

		host = h
	}

	// proxy to docker container logs
	// https://docs.docker.com/reference/api/docker_remote_api_v1.19/#get-container-logs
	client, err := docker.NewClient(host)

	if err != nil {
		return httperr.Server(err)
	}

	quit := make(chan bool)
	logErr := make(chan error)

	go keepAlive(ws, quit)
	go func() {
		e := client.Logs(docker.LogsOptions{
			Container:    fmt.Sprintf("build-%s", build),
			Follow:       true,
			Stdout:       true,
			Stderr:       true,
			Tail:         "all",
			RawTerminal:  false,
			OutputStream: ws,
			ErrorStream:  ws,
		})

		logErr <- e
	}()

ForLoop:
	for {
		select {

		case err = <-logErr:
			break ForLoop

		default:
			b, err := models.Provider().BuildGet(app, build)
			if err != nil {
				break ForLoop
			}

			switch b.Status {
			case "complete":
				err = nil
				break ForLoop
			case "error":
				err = fmt.Errorf("%s build failed", app)
				break ForLoop
			case "failed":
				err = fmt.Errorf("%s build failed", app)
				break ForLoop
			case "timeout":
				err = fmt.Errorf("%s build timeout", app)
				break ForLoop
			}
		}

		time.Sleep(2 * time.Second)
	}

	quit <- true

	return httperr.Server(err)
}

func readImportArtifact(source io.Reader) (*structs.Build, [][]byte, error) {

	var build structs.Build
	var images [][]byte

	gzf, err := gzip.NewReader(source)
	if err != nil {
		return nil, nil, err
	}

	tarReader := tar.NewReader(gzf)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, err
		}

		switch header.Typeflag {
		case tar.TypeReg:
			raw := []byte{}

			if header.Name == "builddata.json" {
				jsonBuf := bytes.NewBuffer(raw)
				io.Copy(jsonBuf, tarReader)

				err = json.Unmarshal(jsonBuf.Bytes(), &build)
				if err != nil {
					return nil, nil, err
				}

			} else {
				if strings.HasSuffix(header.Name, ".tar") {

					buf := bytes.NewBuffer(raw)
					io.Copy(buf, tarReader)

					images = append(images, buf.Bytes())
				}
			}
		default:
			continue
		}
	}

	return &build, images, nil
}

// try to find the docker host that's running a build
// try a few times with a sleep
func findBuildHost(build string) (string, error) {
	for i := 1; i < 5; i++ {
		pss, err := models.ListProcesses(os.Getenv("RACK"))
		if err != nil {
			return "", httperr.Server(err)
		}

		for _, ps := range pss {
			client, err := ps.Docker()
			if err != nil {
				return "", httperr.Server(err)
			}

			res, err := client.ListContainers(docker.ListContainersOptions{
				All: true,
				Filters: map[string][]string{
					"name": []string{fmt.Sprintf("build-%s", build)},
				},
			})

			if len(res) > 0 {
				return fmt.Sprintf("http://%s:2376", ps.Host), nil
			}
		}

		time.Sleep(2 * time.Second)
	}

	return "", fmt.Errorf("could not find build host")
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

// isBuildActive verifies if the build is part of the active release
// Function assumes the build is active if an error occurs to play it safe
func isBuildActive(appName, buildID string) (bool, error) {

	app, err := models.Provider().AppGet(appName)
	if err != nil {
		return true, err
	}

	// To make sure the build exist
	_, err = models.Provider().BuildGet(app.Name, buildID)
	if err != nil {
		return true, err
	}

	if app.Release == "" { // no release means no active build
		return false, nil
	}

	release, err := models.Provider().ReleaseGet(app.Name, app.Release)
	if err != nil {
		return true, err
	}

	return release.Build == buildID, nil
}
