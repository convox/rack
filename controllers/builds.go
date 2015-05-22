package controllers

import (
	"net/http"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/ddollar/logger"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/gorilla/mux"
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
		log.Error(err)
		RenderError(rw, err)
		return
	}

	log.Success("step=build.save app=%q", build.App)

	go build.Execute(repo)

	RenderText(rw, "ok")
}

func buildsLogger(at string) *logger.Logger {
	return logger.New("ns=kernel cn=builds").At(at)
}
