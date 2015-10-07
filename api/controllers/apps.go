package controllers

import (
	"net/http"
	"sort"
	"strings"

	"github.com/convox/rack/api/models"
	"github.com/gorilla/mux"
	"golang.org/x/net/websocket"
)

func AppList(rw http.ResponseWriter, r *http.Request) *HttpError {
	apps, err := models.ListApps()

	if err != nil {
		return ServerError(err)
	}

	sort.Sort(apps)

	return RenderJson(rw, apps)
}

func AppShow(rw http.ResponseWriter, r *http.Request) *HttpError {
	app := mux.Vars(r)["app"]

	a, err := models.GetApp(mux.Vars(r)["app"])

	if awsError(err) == "ValidationError" {
		return HttpErrorf(404, "no such app: %s", app)
	}

	if err != nil && strings.HasPrefix(err.Error(), "no such app") {
		return HttpErrorf(404, "no such app: %s", app)
	}

	if err != nil {
		return ServerError(err)
	}

	return RenderJson(rw, a)
}

func AppCreate(rw http.ResponseWriter, r *http.Request) *HttpError {
	name := r.FormValue("name")

	app := &models.App{
		Name: name,
	}

	err := app.Create()

	if awsError(err) == "AlreadyExistsException" {
		app, err := models.GetApp(name)

		if err != nil {
			return ServerError(err)
		}

		return HttpErrorf(403, "there is already an app named %s (%s)", name, app.Status)
	}

	if err != nil {
		return ServerError(err)
	}

	app, err = models.GetApp(name)

	if err != nil {
		return ServerError(err)
	}

	return RenderJson(rw, app)
}

func AppDelete(rw http.ResponseWriter, r *http.Request) *HttpError {
	name := mux.Vars(r)["app"]

	app, err := models.GetApp(name)

	if awsError(err) == "ValidationError" {
		return HttpErrorf(404, "no such app: %s", name)
	}

	if err != nil {
		return ServerError(err)
	}

	err = app.Delete()

	if err != nil {
		return ServerError(err)
	}

	return RenderSuccess(rw)
}

func AppLogs(ws *websocket.Conn) *HttpError {
	app := mux.Vars(ws.Request())["app"]

	a, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		return HttpErrorf(404, "no such app: %s", app)
	}

	if err != nil {
		return ServerError(err)
	}

	logs := make(chan []byte)
	done := make(chan bool)

	a.SubscribeLogs(logs, done)

	for data := range logs {
		ws.Write(data)
	}

	return nil
}
