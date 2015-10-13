package controllers

import (
	"net/http"
	"sort"

	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/api/models"
	"github.com/gorilla/mux"
	"golang.org/x/net/websocket"
)

func ProcessList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	app := mux.Vars(r)["app"]

	_, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", app)
	}

	processes, err := models.ListProcesses(app)

	if err != nil {
		return httperr.Server(err)
	}

	final := models.Processes{}

	if r.URL.Query().Get("stats") != "false" {
		psch := make(chan models.Process)
		errch := make(chan error)

		for _, p := range processes {
			p := p
			go p.FetchStatsAsync(psch, errch)
		}

		for _, _ = range processes {
			err := <-errch

			if err != nil {
				return httperr.Server(err)
			}

			final = append(final, <-psch)
		}
	} else {
		final = processes
	}

	sort.Sort(final)

	return RenderJson(rw, final)
}

func ProcessShow(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)
	app := vars["app"]
	process := vars["process"]

	_, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", app)
	}

	p, err := models.GetProcess(app, process)

	if err != nil {
		return httperr.Server(err)
	}

	err = p.FetchStats()

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, p)
}

func ProcessExecAttached(ws *websocket.Conn) *httperr.Error {
	vars := mux.Vars(ws.Request())
	app := vars["app"]
	pid := vars["pid"]
	command := ws.Request().Header.Get("Command")

	a, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", app)
	}

	if err != nil {
		return httperr.Server(err)
	}

	return httperr.Server(a.ExecAttached(pid, command, ws))
}

func ProcessRunDetached(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)
	app := vars["app"]
	process := vars["process"]
	command := GetForm(r, "command")

	a, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", app)
	}

	err = a.RunDetached(process, command)

	if err != nil {
		return httperr.Server(err)
	}

	return RenderSuccess(rw)
}

func ProcessRunAttached(ws *websocket.Conn) *httperr.Error {
	vars := mux.Vars(ws.Request())
	app := vars["app"]
	process := vars["process"]
	command := ws.Request().Header.Get("Command")

	a, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", app)
	}

	if err != nil {
		return httperr.Server(err)
	}

	return httperr.Server(a.RunAttached(process, command, ws))
}

func ProcessStop(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)
	app := vars["app"]
	process := vars["process"]

	_, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", app)
	}

	ps, err := models.GetProcess(app, process)

	if err != nil {
		return httperr.Server(err)
	}

	if ps == nil {
		return httperr.Errorf(404, "no such process: %s", process)
	}

	err = ps.Stop()

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, ps)
}
