package controllers

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/provider"
	"github.com/convox/rack/structs"
	"github.com/gorilla/mux"
	"golang.org/x/net/websocket"
)

const StatusCodePrefix = "F1E49A85-0AD7-4AEF-A618-C249C6E6568D:"

// ProcessExecAttached runs an attached command in an existing process
func ProcessExecAttached(ws *websocket.Conn) *httperr.Error {
	vars := mux.Vars(ws.Request())
	header := ws.Request().Header

	app := vars["app"]
	_, err := Provider.AppGet(app)
	if err != nil {
		if provider.ErrorNotFound(err) {
			return httperr.New(404, err)
		}
		return httperr.Server(err)
	}

	pid := vars["pid"]
	command := header.Get("Command")
	height, _ := strconv.Atoi(header.Get("Height"))
	width, _ := strconv.Atoi(header.Get("Width"))

	code, err := Provider.ProcessExec(app, pid, command, structs.ProcessExecOptions{
		Height: height,
		Stream: ws,
		Width:  width,
	})
	if provider.ErrorNotFound(err) {
		return httperr.New(404, err)
	}
	if err != nil {
		return httperr.Server(err)
	}

	if _, err := ws.Write([]byte(fmt.Sprintf("%s%d\n", StatusCodePrefix, code))); err != nil {
		return httperr.Server(err)
	}

	return nil
}

// ProcessGet returns a process for an app
func ProcessGet(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	app := mux.Vars(r)["app"]
	process := mux.Vars(r)["process"]

	ps, err := Provider.ProcessGet(app, process)
	if provider.ErrorNotFound(err) {
		return httperr.NotFound(err)
	}
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, ps)
}

// ProcessList returns a list of processes for an app
func ProcessList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	app := mux.Vars(r)["app"]

	ps, err := Provider.ProcessList(app, structs.ProcessListOptions{})
	if provider.ErrorNotFound(err) {
		return httperr.NotFound(err)
	}
	if err != nil {
		return httperr.Server(err)
	}

	sort.Slice(ps, ps.Less)

	return RenderJson(rw, ps)
}

// ProcessRunAttached runs an attached command in an new process
func ProcessRunAttached(ws *websocket.Conn) *httperr.Error {
	vars := mux.Vars(ws.Request())
	header := ws.Request().Header

	app := vars["app"]
	process := vars["process"]
	command := header.Get("Command")
	release := header.Get("Release")
	height, _ := strconv.Atoi(header.Get("Height"))
	width, _ := strconv.Atoi(header.Get("Width"))

	pid, err := Provider.ProcessRun(app, structs.ProcessRunOptions{
		Command: command,
		Height:  height,
		Width:   width,
		Release: release,
		Service: process,
		Stream:  ws,
	})
	if provider.ErrorNotFound(err) {
		return httperr.New(404, err)
	}
	if err != nil {
		return httperr.Server(err)
	}

	code, err := Provider.ProcessWait(app, pid)
	if err != nil {
		return httperr.Server(err)
	}

	if _, err := ws.Write([]byte(fmt.Sprintf("%s%d\n", StatusCodePrefix, code))); err != nil {
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

	pid, err := Provider.ProcessRun(app, structs.ProcessRunOptions{
		Command: command,
		Release: release,
		Service: process,
	})
	if provider.ErrorNotFound(err) {
		return httperr.New(404, err)
	}
	if err != nil {
		return httperr.Server(err)
	}

	data := map[string]interface{}{"success": true, "pid": pid}

	return RenderJson(rw, data)
}

// ProcessStop stops a Process
func ProcessStop(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)
	app := vars["app"]
	process := vars["process"]

	err := Provider.ProcessStop(app, process)
	if provider.ErrorNotFound(err) {
		return httperr.New(404, err)
	}
	if err != nil {
		return httperr.Server(err)
	}

	return RenderSuccess(rw)
}
