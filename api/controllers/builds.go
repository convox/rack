package controllers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/options"
	"github.com/convox/rack/structs"
	"github.com/gorilla/mux"
	"golang.org/x/net/websocket"
)

func BuildCreate(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)
	app := vars["app"]

	opts := structs.BuildCreateOptions{
		Cache:       options.Bool(!(r.FormValue("cache") == "false")),
		Manifest:    options.String(coalesce(r.FormValue("manifest"), r.FormValue("config"))),
		Description: options.String(r.FormValue("description")),
	}

	if r.FormValue("import") != "" {
		return httperr.Errorf(403, "endpoint deprecated, please update your client")
	}

	image, _, err := r.FormFile("image")
	if err != nil && err != http.ErrMissingFile && err != http.ErrNotMultipart {
		Provider.EventSend("build:create", structs.EventSendOptions{Data: map[string]string{"app": app, "from": "image"}, Error: err.Error()})
		return httperr.Server(err)
	}
	if image != nil {
		build, err := Provider.BuildImport(app, image)
		if err != nil {
			Provider.EventSend("build:create", structs.EventSendOptions{Data: map[string]string{"app": app, "from": "image"}, Error: err.Error()})
			return httperr.Server(err)
		}

		Provider.EventSend("build:create", structs.EventSendOptions{Data: map[string]string{"app": app, "id": build.Id, "from": "image"}})

		return RenderJson(rw, build)
	}

	source, _, err := r.FormFile("source")
	if err != nil && err != http.ErrMissingFile && err != http.ErrNotMultipart {
		Provider.EventSend("build:create", structs.EventSendOptions{Data: map[string]string{"app": app, "from": "source"}, Error: err.Error()})
		return httperr.Server(err)
	}
	if source != nil {
		o, err := Provider.ObjectStore(app, "", source, structs.ObjectStoreOptions{})
		if err != nil {
			Provider.EventSend("build:create", structs.EventSendOptions{Data: map[string]string{"app": app, "from": "source"}, Error: err.Error()})
			return httperr.Server(err)
		}

		build, err := Provider.BuildCreate(app, "tgz", o.Url, opts)
		if err != nil {
			Provider.EventSend("build:create", structs.EventSendOptions{Data: map[string]string{"app": app, "from": "source"}, Error: err.Error()})
			return httperr.Server(err)
		}

		Provider.EventSend("build:create", structs.EventSendOptions{Data: map[string]string{"app": app, "id": build.Id, "from": "source"}})

		return RenderJson(rw, build)
	}

	if index := r.FormValue("index"); index != "" {
		o, err := Provider.ObjectStore(app, "", bytes.NewReader([]byte(index)), structs.ObjectStoreOptions{})
		if err != nil {
			Provider.EventSend("build:create", structs.EventSendOptions{Data: map[string]string{"app": app, "from": "index"}, Error: err.Error()})
			return httperr.Server(err)
		}

		build, err := Provider.BuildCreate(app, "index", o.Url, opts)
		if err != nil {
			Provider.EventSend("build:create", structs.EventSendOptions{Data: map[string]string{"app": app, "from": "index"}, Error: err.Error()})
			return httperr.Server(err)
		}

		Provider.EventSend("build:create", structs.EventSendOptions{Data: map[string]string{"app": app, "id": build.Id, "from": "index"}})

		return RenderJson(rw, build)
	}

	// TODO deprecate
	if repo := r.FormValue("repo"); repo != "" {
		err := fmt.Errorf("repo param has been deprecated")
		return httperr.Server(err)
	}

	if surl := r.FormValue("url"); surl != "" {
		u, err := url.Parse(surl)
		if err != nil {
			Provider.EventSend("build:create", structs.EventSendOptions{Data: map[string]string{"app": app, "from": "url"}, Error: err.Error()})
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
			method = r.FormValue("method")
		default:
			err := httperr.Errorf(403, "unknown extension: %s", ext)
			Provider.EventSend("build:create", structs.EventSendOptions{Data: map[string]string{"app": app, "from": "url"}, Error: err.Error()})
			return err
		}

		build, err := Provider.BuildCreate(app, method, surl, opts)
		if err != nil {
			Provider.EventSend("build:create", structs.EventSendOptions{Data: map[string]string{"app": app, "from": "url"}})
			return httperr.Server(err)
		}

		Provider.EventSend("build:create", structs.EventSendOptions{Data: map[string]string{"app": app, "id": build.Id, "from": "url"}})

		return RenderJson(rw, build)
	}

	err = httperr.Errorf(403, "no build source found")

	Provider.EventSend("build:create", structs.EventSendOptions{Data: map[string]string{"app": app}, Error: err.Error()})

	return httperr.Server(err)
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

func BuildImport(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	app := mux.Vars(r)["app"]
	us := r.FormValue("url")

	if us == "" {
		return httperr.Errorf(403, "must specify url")
	}

	u, err := url.Parse(us)
	if u.Scheme != "object" {
		return httperr.Errorf(403, "only object:// urls are supported")
	}

	or, err := Provider.ObjectFetch(app, u.Path)
	if err != nil {
		return httperr.Server(err)
	}

	b, err := Provider.BuildImport(app, or)
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

	builds, err := Provider.BuildList(app, structs.BuildListOptions{Count: options.Int(limit)})
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

	r, err := Provider.BuildLogs(app, build, structs.LogsOptions{})
	if err != nil {
		return httperr.Server(err)
	}

	io.Copy(ws, r)

	return nil
}

func BuildUpdate(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	v := mux.Vars(r)

	app := v["app"]
	build := v["build"]

	var opts structs.BuildUpdateOptions

	if err := unmarshalOptions(r, &opts); err != nil {
		return httperr.Server(err)
	}

	b, err := Provider.BuildUpdate(app, build, opts)
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, b)
}
