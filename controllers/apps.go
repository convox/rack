package controllers

import (
	"fmt"
	"net/http"
	"sort"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/convox/kernel/Godeps/_workspace/src/golang.org/x/net/websocket"

	"github.com/convox/kernel/models"
)

func AppList(rw http.ResponseWriter, r *http.Request) error {
	apps, err := models.ListApps()

	if err != nil {
		return err
	}

	sort.Sort(apps)

	return RenderJson(rw, apps)
}

func AppShow(rw http.ResponseWriter, r *http.Request) error {
	app := mux.Vars(r)["app"]

	a, err := models.GetApp(mux.Vars(r)["app"])

	if awsError(err) == "ValidationError" {
		return RenderNotFound(rw, fmt.Sprintf("no such app: %s", app))
	}

	if err != nil {
		return err
	}

	return RenderJson(rw, a)
}

func AppCreate(rw http.ResponseWriter, r *http.Request) error {
	name := r.FormValue("name")

	app := &models.App{
		Name: name,
	}

	err := app.Create()

	if awsError(err) == "AlreadyExistsException" {
		app, err := models.GetApp(name)

		if err != nil {
			return err
		}

		return RenderForbidden(rw, fmt.Sprintf("There is already an app named %s (%s)", name, app.Status))
	}

	if err != nil {
		return err
	}

	app, err = models.GetApp(name)

	if err != nil {
		return err
	}

	return RenderJson(rw, app)
}

func AppDelete(rw http.ResponseWriter, r *http.Request) error {
	name := mux.Vars(r)["app"]

	app, err := models.GetApp(name)

	if awsError(err) == "ValidationError" {
		return RenderNotFound(rw, fmt.Sprintf("no such app: %s", name))
	}

	if err != nil {
		return err
	}

	err = app.Delete()

	if err != nil {
		return err
	}

	return RenderSuccess(rw)
}

// func AppDebug(rw http.ResponseWriter, r *http.Request) {
//   log := appsLogger("environment").Start()

//   app := mux.Vars(r)["app"]

//   a, err := models.GetApp(app)

//   if err != nil {
//     helpers.Error(log, err)
//     RenderError(rw, err)
//     return
//   }

//   RenderPartial(rw, "app", "debug", a)
// }

func AppLogs(ws *websocket.Conn) error {
	defer ws.Close()

	app := mux.Vars(ws.Request())["app"]

	a, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		return fmt.Errorf("no such app: %s", app)
	}

	if err != nil {
		return err
	}

	logs := make(chan []byte)
	done := make(chan bool)

	a.SubscribeLogs(logs, done)

	for data := range logs {
		ws.Write(data)
	}

	return nil
}
