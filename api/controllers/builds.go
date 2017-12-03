package controllers

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/structs"
	"github.com/convox/rack/provider"
	"github.com/gorilla/mux"
	"golang.org/x/net/websocket"
)

func BuildCreate(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)
	app := vars["app"]

	opts := structs.BuildOptions{
		Cache:       !(r.FormValue("cache") == "false"),
		Config:      r.FormValue("config"),
		Description: r.FormValue("description"),
	}

	if r.FormValue("import") != "" {
		return httperr.Errorf(403, "endpoint deprecated, please update your client")
	}

	event := &structs.Event{
		Action: "build:create",
		Status: "start",
		Data: map[string]string{
			"app":       app,
			"id":        "n/a",
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}

	image, _, err := r.FormFile("image")
	if err != nil && err != http.ErrMissingFile && err != http.ErrNotMultipart {
		Provider.EventSend(event, err)
		return httperr.Server(err)
	}
	if image != nil {
		build, err := Provider.BuildImport(app, image)
		if err != nil {
			Provider.EventSend(event, err)
			return httperr.Server(err)
		}

		event.Data["id"] = build.Id
		event.Data["from"] = "image"
		Provider.EventSend(event, nil)

		event.Status = "success"
		event.Data["timestamp"] = time.Now().Format(time.RFC3339)
		Provider.EventSend(event, nil)
		return RenderJson(rw, build)
	}

	source, _, err := r.FormFile("source")
	if err != nil && err != http.ErrMissingFile && err != http.ErrNotMultipart {
		Provider.EventSend(event, err)
		return httperr.Server(err)
	}
	if source != nil {
		event.Data["from"] = "source"

		url, err := Provider.ObjectStore("", source, structs.ObjectOptions{})
		if err != nil {
			Provider.EventSend(event, err)
			return httperr.Server(err)
		}

		build, err := Provider.BuildCreate(app, "tgz", url, opts)
		if err != nil {
			Provider.EventSend(event, err)
			return httperr.Server(err)
		}

		event.Data["id"] = build.Id
		Provider.EventSend(event, nil)
		return RenderJson(rw, build)
	}

	if index := r.FormValue("index"); index != "" {
		event.Data["from"] = "index"

		url, err := Provider.ObjectStore("", bytes.NewReader([]byte(index)), structs.ObjectOptions{})
		if err != nil {
			Provider.EventSend(event, err)
			return httperr.Server(err)
		}

		build, err := Provider.BuildCreate(app, "index", url, opts)
		if err != nil {
			Provider.EventSend(event, err)
			return httperr.Server(err)
		}

		event.Data["id"] = build.Id
		Provider.EventSend(event, nil)
		return RenderJson(rw, build)
	}

	// TODO deprecate
	if repo := r.FormValue("repo"); repo != "" {
		err := fmt.Errorf("repo param has been deprecated")
		return httperr.Server(err)
	}

	if surl := r.FormValue("url"); surl != "" {
		event.Data["from"] = "url"

		u, err := url.Parse(surl)
		if err != nil {
			Provider.EventSend(event, err)
			return httperr.Server(err)
		}

		method := ""
		ext := filepath.Ext(u.Path)

		switch ext {
		case ".git":
			method = "git"
		case ".tgz":
			method = "tgz"
		case ".zip":
			method = "zip"
		case "":
			err := httperr.Errorf(403, "building from url requires an extension such as .git")
			Provider.EventSend(event, err)
			return err
		default:
			err := httperr.Errorf(403, "unknown extension: %s", ext)
			Provider.EventSend(event, err)
			return err
		}

		build, err := Provider.BuildCreate(app, method, surl, opts)
		if err != nil {
			Provider.EventSend(event, err)
			return httperr.Server(err)
		}

		event.Data["id"] = build.Id
		Provider.EventSend(event, nil)
		return RenderJson(rw, build)
	}

	err = httperr.Errorf(403, "no build source found")
	Provider.EventSend(event, err)
	return httperr.Server(err)
}

// BuildDelete deletes a build. Makes sure not to delete a build that is contained in the active release
func BuildDelete(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)
	appName := vars["app"]
	buildID := vars["build"]

	app, err := Provider.AppGet(appName)
	if err != nil {
		if provider.ErrorNotFound(err) {
			return httperr.Errorf(404, "no such app: %s", app)
		}

		return httperr.Server(err)
	}

	release, err := Provider.ReleaseGet(app.Name, app.Release)
	if err != nil {
		return httperr.Server(err)
	}

	if release.Build == buildID {
		return httperr.Errorf(400, "cannot delete build of active release: %s", buildID)
	}

	if err := Provider.ReleaseDelete(app.Name, buildID); err != nil {
		return httperr.Server(err)
	}

	build, err := Provider.BuildDelete(app.Name, buildID)
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, build)
}

// BuildExport creates an artifact, representing a build, to be used with another Rack
func BuildExport(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)
	app := vars["app"]
	build := vars["build"]

	b, err := Provider.BuildGet(app, build)
	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", app)
	}
	if err != nil && strings.HasPrefix(err.Error(), "no such build") {
		return httperr.Errorf(404, err.Error())
	}
	if err != nil {
		return httperr.Server(err)
	}

	rw.Header().Set("Content-Type", "application/gzip")
	rw.Header().Set("Transfer-Encoding", "chunked")
	rw.Header().Set("Trailer", "Done")

	if err = Provider.BuildExport(app, b.Id, rw); err != nil {
		return httperr.Server(err)
	}

	rw.Header().Set("Done", "OK")

	return nil
}

func BuildGet(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)
	app := vars["app"]
	build := vars["build"]

	b, err := Provider.BuildGet(app, build)
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

	builds, err := Provider.BuildList(app, int64(limit))
	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", app)
	}
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, builds)
}

func BuildLogs(ws *websocket.Conn) *httperr.Error {
	vars := mux.Vars(ws.Request())

	app := vars["app"]
	build := vars["build"]

	if err := Provider.BuildLogs(app, build, ws); err != nil {
		return httperr.Server(err)
	}

	return nil
}
