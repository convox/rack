package controllers

import (
	"net/http"
	"sort"
	"strconv"
	"sync"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/convox/rack/Godeps/_workspace/src/golang.org/x/net/websocket"
	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/api/models"
)

func ProcessList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	app := mux.Vars(r)["app"]
	stats := r.URL.Query().Get("stats") == "true"

	_, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", app)
	}

	processes, err := models.ListProcesses(app)

	if err != nil {
		return httperr.Server(err)
	}

	if stats {
		w := new(sync.WaitGroup)
		erch := make(chan error, len(processes))

		for _, p := range processes {
			w.Add(1)

			go func(p *models.Process, w *sync.WaitGroup, erch chan error) {
				err := p.FetchStats()

				w.Done()

				if err != nil {
					erch <- err
				}
			}(p, w, erch)
		}

		w.Wait()

		select {
		case err := <-erch:
			return httperr.Server(err)
		default:
			// noop
		}
	}

	sort.Sort(models.Processes(processes))

	return RenderJson(rw, processes)
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
	header := ws.Request().Header

	app := vars["app"]
	pid := vars["pid"]
	command := header.Get("Command")
	height, _ := strconv.Atoi(header.Get("Height"))
	width, _ := strconv.Atoi(header.Get("Width"))

	a, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", app)
	}

	if err != nil {
		return httperr.Server(err)
	}

	return httperr.Server(a.ExecAttached(pid, command, height, width, ws))
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
	header := ws.Request().Header

	app := vars["app"]
	process := vars["process"]
	command := header.Get("Command")
	height, _ := strconv.Atoi(header.Get("Height"))
	width, _ := strconv.Atoi(header.Get("Width"))

	a, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", app)
	}

	if err != nil {
		return httperr.Server(err)
	}

	return httperr.Server(a.RunAttached(process, command, height, width, ws))
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
