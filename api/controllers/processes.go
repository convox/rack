package controllers

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/api/models"
	"github.com/convox/rack/api/structs"
	"github.com/convox/rack/provider"
	"github.com/gorilla/mux"
	"golang.org/x/net/websocket"
)

func ProcessList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	app := mux.Vars(r)["app"]
	// stats := r.URL.Query().Get("stats") == "true"

	ps, err := models.Provider().ProcessList(app)
	if provider.ErrorNotFound(err) {
		return httperr.Errorf(404, app)
	}
	if err != nil {
		return httperr.Server(err)
	}

	sort.Sort(ps)

	return RenderJson(rw, ps)
}

func ProcessExecAttached(ws *websocket.Conn) *httperr.Error {
	vars := mux.Vars(ws.Request())
	header := ws.Request().Header

	app := vars["app"]
	pid := vars["pid"]
	command := header.Get("Command")
	height, _ := strconv.Atoi(header.Get("Height"))
	width, _ := strconv.Atoi(header.Get("Width"))

	fmt.Printf("height = %+v\n", height)
	fmt.Printf("width = %+v\n", width)

	err := models.Provider().ProcessExec(app, pid, command, ws)
	if provider.ErrorNotFound(err) {
		return httperr.New(404, err)
	}
	if err != nil {
		return httperr.Server(err)
	}

	return nil
}

func ProcessRunDetached(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)
	app := vars["app"]
	process := vars["process"]
	command := GetForm(r, "command")
	release := GetForm(r, "release")

	ps, err := models.Provider().ProcessRun(app, process, structs.ProcessRunOptions{
		Command: command,
		Release: release,
	})
	fmt.Printf("ps = %+v\n", ps)
	fmt.Printf("err = %+v\n", err)
	if provider.ErrorNotFound(err) {
		return httperr.New(404, err)
	}
	if err != nil {
		return httperr.Server(err)
	}

	return RenderSuccess(rw)
}

func ProcessRunAttached(ws *websocket.Conn) *httperr.Error {
	vars := mux.Vars(ws.Request())
	header := ws.Request().Header

	app := vars["app"]
	process := vars["process"]
	command := header.Get("Command")
	release := header.Get("Release")
	// height, _ := strconv.Atoi(header.Get("Height"))
	// width, _ := strconv.Atoi(header.Get("Width"))

	opts := structs.ProcessRunOptions{
		Command: command,
		Release: release,
	}

	ps, err := models.Provider().ProcessRun(app, process, opts)
	fmt.Printf("ps = %+v\n", ps)
	fmt.Printf("err = %+v\n", err)
	if err != nil {
		return httperr.Server(err)
	}

	err = models.Provider().ProcessAttach(ps.ID, ws)
	if err != nil {
		return httperr.Server(err)
	}

	return nil
}

func ProcessStop(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)
	app := vars["app"]
	process := vars["process"]

	err := models.Provider().ProcessStop(app, process)
	if provider.ErrorNotFound(err) {
		return httperr.New(404, err)
	}
	if err != nil {
		return httperr.Server(err)
	}

	return RenderSuccess(rw)
}
