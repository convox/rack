package controllers

import (
	"fmt"
	"net/http"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/ddollar/logger"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/gorilla/websocket"

	"github.com/convox/kernel/helpers"
	"github.com/convox/kernel/models"
)

func init() {
	RegisterPartial("process", "logs")
	RegisterPartial("process", "resources")

	RegisterTemplate("process", "layout", "process")
}

func ProcessShow(rw http.ResponseWriter, r *http.Request) {
	log := processesLogger("show").Start()

	vars := mux.Vars(r)

	process, err := models.GetProcess(vars["app"], vars["process"])

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	RenderTemplate(rw, "process", process)
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

func ProcessStream(rw http.ResponseWriter, r *http.Request) {
	log := processesLogger("stream").Start()

	vars := mux.Vars(r)

	process, err := models.GetProcess(vars["app"], vars["process"])

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	logs := make(chan []byte)
	done := make(chan bool)

	process.SubscribeLogs(logs, done)

	log.At("upgrade")
	ws, err := upgrader.Upgrade(rw, r, nil)

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	defer ws.Close()

	for data := range logs {
		ws.WriteMessage(websocket.TextMessage, data)
	}

	fmt.Println("ended")
}

func ProcessResources(rw http.ResponseWriter, r *http.Request) {
	log := processesLogger("resources").Start()

	vars := mux.Vars(r)
	app := vars["app"]
	process := vars["process"]

	resources, err := models.ListProcessResources(app, process)

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	RenderPartial(rw, "process", "resources", resources)
}

func processesLogger(at string) *logger.Logger {
	return logger.New("ns=kernel cn=processes").At(at)
}
