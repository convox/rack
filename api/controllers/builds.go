package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/api/models"
	"github.com/convox/rack/api/structs"
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

	return RenderJson(rw, b)
}

func BuildCreate(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)
	app := vars["app"]

	opts := structs.BuildOptions{
		Cache:       !(r.FormValue("cache") == "false"),
		Config:      r.FormValue("config"),
		Description: r.FormValue("description"),
	}

	fmt.Printf("app = %+v\n", app)
	fmt.Printf("opts = %+v\n", opts)

	image, _, err := r.FormFile("image")
	if err != nil && err != http.ErrMissingFile && err != http.ErrNotMultipart {
		return httperr.Server(err)
	}
	if image != nil {
	}

	source, _, err := r.FormFile("source")
	if err != nil && err != http.ErrMissingFile && err != http.ErrNotMultipart {
		return httperr.Server(err)
	}
	if source != nil {
		url, err := models.Provider().ObjectStore("", source, structs.ObjectOptions{})
		if err != nil {
			return httperr.Server(err)
		}

		build, err := models.Provider().BuildCreate(app, "tgz", url, opts)
		if err != nil {
			return httperr.Server(err)
		}

		return RenderJson(rw, build)
	}

	if index := r.FormValue("index"); index != "" {
	}

	if repo := r.FormValue("repo"); repo != "" {
	}

	return httperr.Errorf(403, "no build source found")

	// test
	// // Log into private registries that we might pull from
	// // TODO: move to prodiver BuildCreate
	// err = models.LoginPrivateRegistries()
	// if err != nil {
	//   return httperr.Server(err)
	// }

	// a, err := models.Provider().AppGet(app)
	// if err != nil {
	//   return httperr.Server(err)
	// }

	// // Log into registry that we will push to
	// _, err = models.AppDockerLogin(*a)
	// if err != nil {
	//   return httperr.Server(err)
	// }

	// var b *structs.Build

	// if source != nil {

	//   if buildImport {
	//     b, err = models.Provider().BuildImport(a.Name, source)
	//   } else {
	//     // if source file was posted, build from tar
	//     b, err = models.Provider().BuildCreateTar(app, source, r.FormValue("manifest"), r.FormValue("description"), cache)
	//   }

	// } else if repo != "" {
	//   b, err = models.Provider().BuildCreateRepo(app, repo, r.FormValue("manifest"), r.FormValue("description"), cache)
	// } else if index != "" {
	//   var i structs.Index
	//   err := json.Unmarshal([]byte(index), &i)
	//   if err != nil {
	//     return httperr.Server(err)
	//   }

	//   b, err = models.Provider().BuildCreateIndex(app, i, manifest, description, cache)
	// } else {
	//   return httperr.Errorf(403, "no source, repo or index")
	// }

	// if err != nil {
	//   return httperr.Server(err)
	// }

	// return RenderJson(rw, b)
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

// BuildExport creates an artifact, representing a build, to be used with another Rack
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

	rw.Header().Set("Content-Type", "application/octet-stream")

	if err = models.Provider().BuildExport(app, b.Id, rw); err != nil {
		return httperr.Server(err)
	}

	return nil
}

func BuildLogs(ws *websocket.Conn) *httperr.Error {
	vars := mux.Vars(ws.Request())

	app := vars["app"]
	build := vars["build"]

	if err := models.Provider().BuildLogs(app, build, ws); err != nil {
		return httperr.Server(err)
	}

	return nil
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
