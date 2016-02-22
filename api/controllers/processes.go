package controllers

import (
	"net/http"
	"sort"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/convox/rack/Godeps/_workspace/src/golang.org/x/net/websocket"
	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/api/provider"
	"github.com/convox/rack/api/structs"
)

func ProcessList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	app := mux.Vars(r)["app"]

	_, err := provider.AppGet(app)

	if err != nil {
		return httperr.Server(err)
	}

	processes, err := provider.ProcessList(app)

	if err != nil {
		return httperr.Server(err)
	}

	final := structs.Processes{}

	if r.URL.Query().Get("stats") != "false" {
		psch := make(chan structs.Process)
		errch := make(chan error)

		for _, p := range processes {
			go func(p structs.Process) {
				stats, err := provider.ProcessStats(app, p.Id)

				if err != nil {
					errch <- err
					return
				}

				p.Cpu = stats.Cpu
				p.Memory = stats.Memory

				psch <- p
			}(p)
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
	pid := vars["pid"]

	_, err := provider.AppGet(app)

	if err != nil {
		return httperr.Server(err)
	}

	p, err := provider.ProcessGet(app, pid)

	if err != nil {
		return httperr.Server(err)
	}

	stats, err := provider.ProcessStats(app, pid)

	if err != nil {
		return httperr.Server(err)
	}

	p.Cpu = stats.Cpu
	p.Memory = stats.Memory

	return RenderJson(rw, p)
}

func ProcessExecAttached(ws *websocket.Conn) *httperr.Error {
	vars := mux.Vars(ws.Request())
	app := vars["app"]
	pid := vars["pid"]
	command := ws.Request().Header.Get("Command")

	return httperr.Server(provider.ProcessExec(app, pid, command, ws))
}

func ProcessRunDetached(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)
	app := vars["app"]
	process := vars["process"]
	command := GetForm(r, "command")

	err := provider.RunDetached(app, process, command)

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

	return httperr.Server(provider.RunAttached(app, process, command, ws))
}

func ProcessStop(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)
	app := vars["app"]
	pid := vars["pid"]

	ps, err := provider.ProcessGet(app, pid)

	if err != nil {
		return httperr.Server(err)
	}

	if ps == nil {
		return httperr.Errorf(404, "no such process: %s", pid)
	}

	err = provider.ProcessStop(app, pid)

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, ps)
}
