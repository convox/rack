package controllers

import (
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/api/models"
	"github.com/convox/rack/api/provider"
	"github.com/convox/rack/api/structs"
	"github.com/gorilla/mux"
	"golang.org/x/net/websocket"
)

func AppList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	apps, err := models.ListApps()

	if err != nil {
		return httperr.Server(err)
	}

	sort.Sort(apps)

	return RenderJson(rw, apps)
}

func AppShow(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	app := mux.Vars(r)["app"]

	a, err := models.GetApp(mux.Vars(r)["app"])

	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", app)
	}

	if err != nil && strings.HasPrefix(err.Error(), "no such app") {
		return httperr.Errorf(404, "no such app: %s", app)
	}

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, a)
}

func AppCreate(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	name := r.FormValue("name")

	// Early check for unbound app only.
	if app, err := models.GetAppUnbound(name); err == nil {
		return httperr.Errorf(403, "there is already a legacy app named %s (%s). We recommend you delete this app and create it again.", name, app.Status)
	}

	// If unbound check fails this will result in a bound app.
	app := &models.App{Name: name}
	err := app.Create()

	if awsError(err) == "AlreadyExistsException" {
		app, err := models.GetApp(name)

		if err != nil {
			return httperr.Server(err)
		}

		return httperr.Errorf(403, "there is already an app named %s (%s)", name, app.Status)
	}

	if err != nil {
		return httperr.Server(err)
	}

	app, err = models.GetApp(name)

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, app)
}

func AppDelete(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	name := mux.Vars(r)["app"]

	app, err := models.GetApp(name)

	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", name)
	}

	if err != nil {
		return httperr.Server(err)
	}

	if app.Tags["Type"] != "app" || app.Tags["System"] != "convox" || app.Tags["Rack"] != os.Getenv("RACK") {
		return httperr.Errorf(404, "invalid app: %s", name)
	}

	err = app.Delete()

	if err != nil {
		return httperr.Server(err)
	}

	return RenderSuccess(rw)
}

func AppLogs(ws *websocket.Conn) *httperr.Error {
	app := mux.Vars(ws.Request())["app"]
	header := ws.Request().Header

	follow := true
	if header.Get("Follow") == "false" {
		follow = false
	}

	since, err := time.ParseDuration(header.Get("Since"))
	if err != nil {
		return httperr.Errorf(403, "Invalid duration %s", header.Get("Since"))
	}

	err = provider.LogStream(app, ws, structs.LogStreamOptions{
		Filter: header.Get("Filter"),
		Follow: follow,
		Since:  since,
	})
	if err != nil {
		if strings.HasSuffix(err.Error(), "write: broken pipe") {
			return nil
		}
		return httperr.Server(err)
	}
	return nil
}
