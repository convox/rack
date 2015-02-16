package controllers

import (
	"fmt"
	"net/http"

	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/gorilla/websocket"

	"github.com/convox/kernel/web/models"
)

func init() {
	RegisterTemplate("process", "layout", "process")
}

func ProcessShow(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	process, err := models.GetProcess(vars["app"], vars["process"])

	if err != nil {
		RenderError(rw, err)
		return
	}

	RenderTemplate(rw, "process", process)
}

func ProcessLogs(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	process, err := models.GetProcess(vars["app"], vars["process"])

	if err != nil {
		RenderError(rw, err)
		return
	}

	logs := make(chan []byte)
	done := make(chan bool)

	process.SubscribeLogs(logs, done)

	ws, err := upgrader.Upgrade(rw, r, nil)

	if err != nil {
		RenderError(rw, err)
		return
	}

	defer ws.Close()

	for data := range logs {
		ws.WriteMessage(websocket.TextMessage, data)
	}

	fmt.Println("ended")
}
