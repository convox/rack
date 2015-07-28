package controllers

import (
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/ddollar/logger"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/convox/kernel/Godeps/_workspace/src/golang.org/x/net/websocket"

	"github.com/convox/kernel/helpers"
	"github.com/convox/kernel/models"
)

func init() {
	RegisterPartial("process", "logs")
	RegisterPartial("process", "resources")

	RegisterTemplate("process", "layout", "process")
}

func ProcessList(rw http.ResponseWriter, r *http.Request) {
	log := appsLogger("processes").Start()

	app := mux.Vars(r)["app"]

	_, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		RenderNotFound(rw, fmt.Sprintf("no such app: %s", app))
		return
	}

	processes, err := models.ListProcesses(app)

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	RenderJson(rw, processes)
}

func ProcessShow(rw http.ResponseWriter, r *http.Request) {
	log := processesLogger("show").Start()

	vars := mux.Vars(r)
	app := vars["app"]
	process := vars["process"]

	_, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		RenderNotFound(rw, fmt.Sprintf("no such app: %s", app))
		return
	}

	p, err := models.GetProcess(app, process)

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	RenderTemplate(rw, "process", p)
}

func ProcessLogs(rw http.ResponseWriter, r *http.Request) {
	// log := processesLogger("logs").Start()

	vars := mux.Vars(r)
	app := vars["app"]
	process := vars["process"]

	params := map[string]string{
		"App":     app,
		"Process": process,
	}

	RenderPartial(rw, "process", "logs", params)
}

func ProcessRun(rw http.ResponseWriter, r *http.Request) {
	log := processesLogger("run").Start()

	vars := mux.Vars(r)
	app := vars["app"]
	process := vars["process"]
	command := GetForm(r, "command")

	_, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		RenderNotFound(rw, fmt.Sprintf("no such app: %s", app))
		return
	}

	ps, err := models.GetProcess(app, process)

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	if ps == nil {
		RenderNotFound(rw, fmt.Sprintf("no such process: %s", app, process))
		return
	}

	err = ps.Run(models.ProcessRunOptions{
		Command: command,
	})

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	RenderText(rw, "ok")
}

func ProcessRunAttached(ws *websocket.Conn) {
	defer ws.Close()

	log := processesLogger("run.attached").Start()

	vars := mux.Vars(ws.Request())
	app := vars["app"]
	process := vars["process"]
	command := ws.Request().Header.Get("Command")

	ps, err := models.GetProcess(app, process)

	if err != nil {
		helpers.Error(log, err)
		ws.Write([]byte(fmt.Sprintf("error: %s\n", err)))
		return
	}

	log.Success("step=upgrade app=%q", ps.App)

	defer ws.Close()

	err = ps.RunAttached(command, ws)

	if err != nil {
		helpers.Error(log, err)
		ws.Write([]byte(fmt.Sprintf("error: %s\n", err)))
		return
	}

	log.Success("step=ended app=%q", ps.App)
}

func copyWait(w io.Writer, r io.Reader, wg *sync.WaitGroup) {
	io.Copy(w, r)
	wg.Done()
}

func ProcessStop(rw http.ResponseWriter, r *http.Request) {
	log := processesLogger("stop").Start()

	vars := mux.Vars(r)
	app := vars["app"]
	id := vars["id"]

	_, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		RenderNotFound(rw, fmt.Sprintf("no such app: %s", app))
		return
	}

	ps, err := models.GetProcessById(app, id)

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	if ps == nil {
		RenderNotFound(rw, fmt.Sprintf("no such process: %s", id))
		return
	}

	err = ps.Stop()

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	RenderText(rw, "ok")
}

func ProcessTop(rw http.ResponseWriter, r *http.Request) {
	log := processesLogger("info").Start()

	vars := mux.Vars(r)
	app := vars["app"]
	id := vars["id"]

	_, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		RenderNotFound(rw, fmt.Sprintf("no such app: %s", app))
		return
	}

	ps, err := models.GetProcessById(app, id)

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	if ps == nil {
		RenderNotFound(rw, fmt.Sprintf("no such process: %s", id))
		return
	}

	info, err := ps.Top()

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	RenderJson(rw, info)
}

func processesLogger(at string) *logger.Logger {
	return logger.New("ns=kernel cn=processes").At(at)
}
