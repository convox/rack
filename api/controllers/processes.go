package controllers

import (
	"net/http"
	"sort"

	"github.com/convox/rack/api/models"
	"github.com/gorilla/mux"
	"golang.org/x/net/websocket"
)

func ProcessList(rw http.ResponseWriter, r *http.Request) *HttpError {
	app := mux.Vars(r)["app"]

	_, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		return HttpErrorf(404, "no such app: %s", app)
	}

	processes, err := models.ListProcesses(app)

	if err != nil {
		return ServerError(err)
	}

	final := models.Processes{}
	psch := make(chan models.Process)
	errch := make(chan error)

	for _, p := range processes {
		p := p
		go p.FetchStatsAsync(psch, errch)
	}

	for _, _ = range processes {
		err := <-errch

		if err != nil {
			return ServerError(err)
		}

		final = append(final, <-psch)
	}

	sort.Sort(final)

	return RenderJson(rw, final)
}

func ProcessShow(rw http.ResponseWriter, r *http.Request) *HttpError {
	vars := mux.Vars(r)
	app := vars["app"]
	process := vars["process"]

	_, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		return HttpErrorf(404, "no such app: %s", app)
	}

	p, err := models.GetProcess(app, process)

	if err != nil {
		return ServerError(err)
	}

	err = p.FetchStats()

	if err != nil {
		return ServerError(err)
	}

	return RenderJson(rw, p)
}

func ProcessExecAttached(ws *websocket.Conn) *HttpError {
	vars := mux.Vars(ws.Request())
	app := vars["app"]
	pid := vars["pid"]
	command := ws.Request().Header.Get("Command")

	a, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		return HttpErrorf(404, "no such app: %s", app)
	}

	if err != nil {
		return ServerError(err)
	}

	return ServerError(a.ExecAttached(pid, command, ws))
}

func ProcessRunDetached(rw http.ResponseWriter, r *http.Request) *HttpError {
	vars := mux.Vars(r)
	app := vars["app"]
	process := vars["process"]
	command := GetForm(r, "command")

	a, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		return HttpErrorf(404, "no such app: %s", app)
	}

	err = a.RunDetached(process, command)

	if err != nil {
		return ServerError(err)
	}

	return RenderSuccess(rw)
}

func ProcessRunAttached(ws *websocket.Conn) *HttpError {
	vars := mux.Vars(ws.Request())
	app := vars["app"]
	process := vars["process"]
	command := ws.Request().Header.Get("Command")

	a, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		return HttpErrorf(404, "no such app: %s", app)
	}

	if err != nil {
		return ServerError(err)
	}

	return ServerError(a.RunAttached(process, command, ws))
}

func ProcessStop(rw http.ResponseWriter, r *http.Request) *HttpError {
	vars := mux.Vars(r)
	app := vars["app"]
	process := vars["process"]

	_, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		return HttpErrorf(404, "no such app: %s", app)
	}

	ps, err := models.GetProcess(app, process)

	if err != nil {
		return ServerError(err)
	}

	if ps == nil {
		return HttpErrorf(404, "no such process: %s", process)
	}

	err = ps.Stop()

	if err != nil {
		return ServerError(err)
	}

	return RenderJson(rw, ps)
}
