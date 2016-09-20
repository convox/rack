package controllers

import (
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

	ps, err := models.Provider().ProcessList(app)
	if provider.ErrorNotFound(err) {
		return httperr.NotFound(err)
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

	err := models.Provider().ProcessExec(app, pid, command, ws, structs.ProcessExecOptions{
		Height: height,
		Width:  width,
	})
	if provider.ErrorNotFound(err) {
		return httperr.New(404, err)
	}
	if err != nil {
		return httperr.Server(err)
	}

	return nil
}

func ProcessRunAttached(ws *websocket.Conn) *httperr.Error {
	vars := mux.Vars(ws.Request())
	header := ws.Request().Header

	app := vars["app"]
	process := vars["process"]
	command := header.Get("Command")
	release := header.Get("Release")
	height, _ := strconv.Atoi(header.Get("Height"))
	width, _ := strconv.Atoi(header.Get("Width"))

	_, err := models.Provider().ProcessRun(app, process, structs.ProcessRunOptions{
		Command: command,
		Height:  height,
		Width:   width,
		Release: release,
		Stream:  ws,
	})
	if provider.ErrorNotFound(err) {
		return httperr.New(404, err)
	}
	if err != nil {
		return httperr.Server(err)
	}

	return nil
}

// ProcessRunDetached runs a process in the background
func ProcessRunDetached(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)
	app := vars["app"]
	process := vars["process"]
	command := GetForm(r, "command")
	release := GetForm(r, "release")

	_, err := models.Provider().ProcessRun(app, process, structs.ProcessRunOptions{
		Command: command,
		Release: release,
	})
	if provider.ErrorNotFound(err) {
		return httperr.New(404, err)
	}
	if err != nil {
		return httperr.Server(err)
	}

	return RenderSuccess(rw)
}

// ProcessStop stops a Process
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
