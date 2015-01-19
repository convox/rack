package controllers

import (
	"fmt"
	"math/rand"
	"net/http"

	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/convox/kernel/web/models"
)

func init() {
	RegisterTemplate("process", "layout", "process")
}

func ProcessShow(rw http.ResponseWriter, r *http.Request) {
	cluster := models.Cluster{Name: mux.Vars(r)["cluster"]}
	app := models.App{mux.Vars(r)["app"], "unknown", 40, 100, 60, 100, 10, 100, &cluster, make(models.Processes, 2)}
	process := models.Process{mux.Vars(r)["process"], "bundle exec rails start -p $PORT", 3, 60, 100, 20, 100, 5, 100, &app, make(models.Containers, (rand.Int()%10)+2)}
	for i := 0; i < len(process.Containers); i++ {
		container := models.Container{Name: fmt.Sprintf("%s.%d", mux.Vars(r)["process"], i+1)}
		container.CpuUsed = (rand.Int() % 100)
		container.CpuTotal = 100
		container.MemoryUsed = rand.Int() % 100
		container.MemoryTotal = 100
		container.DiskUsed = rand.Int() % 100
		container.DiskTotal = 100
		process.Containers[i] = container
	}
	RenderTemplate(rw, "process", process)
}
