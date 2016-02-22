package controllers

import (
	"net/http"
	"sort"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/convox/rack/Godeps/_workspace/src/golang.org/x/net/websocket"
	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/api/models"
	"github.com/convox/rack/api/provider"
)

func AppList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	apps, err := provider.AppList()

	if err != nil {
		return httperr.Server(err)
	}

	sort.Sort(apps)

	return RenderJson(rw, apps)
}

func AppShow(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	a, err := provider.AppGet(mux.Vars(r)["app"])

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, a)
}

func AppCreate(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	name := r.FormValue("name")

	err := provider.AppCreate(name)

	if err != nil {
		return httperr.Server(err)
	}

	app, err := provider.AppGet(name)

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, app)
}

func AppDelete(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	name := mux.Vars(r)["app"]

	app, err := provider.AppGet(name)

	if err != nil {
		return httperr.Server(err)
	}

	err = provider.AppDelete(app)

	if err != nil {
		return httperr.Server(err)
	}

	return RenderSuccess(rw)
}

func AppLogs(ws *websocket.Conn) *httperr.Error {
	app := mux.Vars(ws.Request())["app"]

	a, err := provider.AppGet(app)

	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", app)
	}

	if err != nil {
		return httperr.Server(err)
	}

	logs := make(chan []byte)
	done := make(chan bool)

	go models.SubscribeKinesis(a.Outputs["Kinesis"], logs, done)
	go signalWsClose(ws, done)

	for data := range logs {
		ws.Write(data)
	}

	return nil
}
