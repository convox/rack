package controllers

import (
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/api/models"
	"github.com/convox/rack/api/structs"
	"github.com/convox/rack/provider"
	"github.com/gorilla/mux"
	"golang.org/x/net/websocket"
)

// AppList lists installed apps
func AppList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	apps, err := models.ListApps()
	if err != nil {
		return httperr.Server(err)
	}

	sort.Sort(apps)

	return RenderJson(rw, apps)
}

// AppGet gets app information
func AppGet(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	app := mux.Vars(r)["app"]

	if app == os.Getenv("RACK") {
		return httperr.Errorf(404, "rack %s is not an app", app)
	}

	a, err := models.Provider().AppGet(app)
	if err != nil {
		if provider.ErrorNotFound(err) {
			return httperr.Errorf(404, "no such app: %s", app)
		}

		return httperr.Server(err)
	}

	return RenderJson(rw, a)
}

// AppCancel cancels an app update
func AppCancel(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	app := mux.Vars(r)["app"]

	err := models.Provider().AppCancel(app)
	if provider.ErrorNotFound(err) {
		return httperr.NotFound(err)
	}
	if err != nil {
		return httperr.Server(err)
	}

	return RenderSuccess(rw)
}

// AppCreate creates an application
func AppCreate(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	name := r.FormValue("name")

	if name == os.Getenv("RACK") {
		return httperr.Errorf(403, "application name cannot match rack name (%s). Please choose a different name for your app.", name)
	}

	switch r.FormValue("generation") {
	case "2":
		return appCreateGeneration2(rw, r)
	default:
		return appCreateGeneration1(rw, r)
	}

	return httperr.Server(fmt.Errorf("unknown generation"))
}

func appCreateGeneration1(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	name := r.FormValue("name")

	app := &models.App{Name: name}

	err := app.Create()
	if awsError(err) == "AlreadyExistsException" {
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

func appCreateGeneration2(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	name := r.FormValue("name")

	app, err := models.Provider().AppCreate(name)
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, app)
}

// AppDelete deletes an application
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

// AppLogs show an app's logs
func AppLogs(ws *websocket.Conn) *httperr.Error {
	app := mux.Vars(ws.Request())["app"]
	header := ws.Request().Header

	var err error

	follow := true
	if header.Get("Follow") == "false" {
		follow = false
	}

	since := 2 * time.Minute
	if s := header.Get("Since"); s != "" {
		since, err = time.ParseDuration(s)
		if err != nil {
			return httperr.Errorf(403, "Invalid duration %s", s)
		}
	}

	err = models.Provider().LogStream(app, ws, structs.LogStreamOptions{
		Filter: header.Get("Filter"),
		Follow: follow,
		Since:  time.Now().Add(-1 * since),
	})
	if err != nil {
		if strings.HasSuffix(err.Error(), "write: broken pipe") {
			return nil
		}
		return httperr.Server(err)
	}
	return nil
}
