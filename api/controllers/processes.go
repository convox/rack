package controllers

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/options"
	"github.com/convox/rack/provider"
	"github.com/convox/rack/structs"
	"github.com/gorilla/mux"
	"golang.org/x/net/websocket"
)

const StatusCodePrefix = "F1E49A85-0AD7-4AEF-A618-C249C6E6568D:"

// ProcessExecAttached runs an attached command in an existing process
func ProcessExecAttached(ws *websocket.Conn) *httperr.Error {
	vars := mux.Vars(ws.Request())

	app := vars["app"]
	_, err := Provider.AppGet(app)
	if err != nil {
		if provider.ErrorNotFound(err) {
			return httperr.New(404, err)
		}
		return httperr.Server(err)
	}

	h := ws.Request().Header

	pid := vars["pid"]
	command := h.Get("Command")

	opts := structs.ProcessExecOptions{
		Stream: ws,
	}

	if v := h.Get("Height"); v != "" {
		i, err := strconv.Atoi(v)
		if err != nil {
			return httperr.Server(fmt.Errorf("height must be numeric"))
		}
		opts.Height = aws.Int(i)
	}

	if v := h.Get("Width"); v != "" {
		i, err := strconv.Atoi(v)
		if err != nil {
			return httperr.Server(fmt.Errorf("width must be numeric"))
		}
		opts.Width = aws.Int(i)
	}

	code, err := Provider.ProcessExec(app, pid, command, opts)
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
	h := ws.Request().Header

	app := vars["app"]

	opts := structs.ProcessRunOptions{
		Service: options.String(vars["process"]),
		Stream:  ws,
	}

	if v := h.Get("Command"); v != "" {
		opts.Command = options.String(v)
	}

	if v := h.Get("Release"); v != "" {
		opts.Release = options.String(v)
	}

	if v := h.Get("Height"); v != "" {
		i, err := strconv.Atoi(v)
		if err != nil {
			return httperr.Server(fmt.Errorf("height must be numeric"))
		}
		opts.Height = aws.Int(i)
	}

	if v := h.Get("Width"); v != "" {
		i, err := strconv.Atoi(v)
		if err != nil {
			return httperr.Server(fmt.Errorf("width must be numeric"))
		}
		opts.Width = aws.Int(i)
	}

	pid, err := Provider.ProcessRun(app, opts)
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

	opts := structs.ProcessRunOptions{
		Service: options.String(vars["process"]),
	}

	if v := GetForm(r, "command"); v != "" {
		opts.Command = options.String(v)
	}

	if v := GetForm(r, "release"); v != "" {
		opts.Release = options.String(v)
	}

	pid, err := Provider.ProcessRun(app, opts)
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
