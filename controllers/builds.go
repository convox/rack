package controllers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/ddollar/logger"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/gorilla/websocket"

	"github.com/convox/kernel/helpers"
	"github.com/convox/kernel/models"
)

func BuildCreate(rw http.ResponseWriter, r *http.Request) {
	log := buildsLogger("create").Start()

	vars := mux.Vars(r)
	app := vars["app"]
	repo := GetForm(r, "repo")

	build := models.NewBuild(app)

	err := build.Save()

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	log.Success("step=build.save app=%q", build.App)

	go build.Execute(repo)

	RenderText(rw, "ok")
}

func BuildLogs(rw http.ResponseWriter, r *http.Request) {
	log := buildsLogger("logs").Start()

	vars := mux.Vars(r)
	app := vars["app"]
	id := vars["build"]

	build, err := models.GetBuild(app, id)

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	log.Success("step=build.logs app=%q", build.App)

	RenderText(rw, build.Logs)
}

func BuildStatus(rw http.ResponseWriter, r *http.Request) {
	log := buildsLogger("status").Start()

	vars := mux.Vars(r)
	app := vars["app"]
	id := vars["build"]

	build, err := models.GetBuild(app, id)

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	RenderText(rw, build.Status)
}

func BuildStream(rw http.ResponseWriter, r *http.Request) {
	log := buildsLogger("stream").Start()

	vars := mux.Vars(r)
	app := vars["app"]
	id := vars["build"]

	b, err := models.GetBuild(app, id)

	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	ws, err := upgrader.Upgrade(rw, r, nil)

	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	log.Success("step=upgrade build=%q", b.Id)

	defer ws.Close()

	ws.WriteMessage(websocket.TextMessage, []byte(b.Logs))

	if b.Status == "failed" || b.Status == "complete" {
		log.Success("step=ended build=%q", b.Id)
		return
	}

	// Every 2 seconds check for new logs and write to websocket
	ticker := time.NewTicker(2 * time.Second)
	quit := make(chan struct{})
	logs := b.Logs

	go func() {
		for {
			select {
			case <-ticker.C:
				b, err := models.GetBuild(app, id)

				if err != nil {
					helpers.Error(log, err)
					RenderError(rw, err)
					return
				}

				if b.Logs != logs {
					latest := strings.TrimPrefix(b.Logs, logs)
					ws.WriteMessage(websocket.TextMessage, []byte(latest))
					logs = b.Logs
				}

				if b.Status == "failed" || b.Status == "complete" {
					log.Success("step=ended build=%q", b.Id)
					ticker.Stop()
					return
				}
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()

	<-quit
}

func buildsLogger(at string) *logger.Logger {
	return logger.New("ns=kernel cn=builds").At(at)
}
