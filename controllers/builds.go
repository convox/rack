package controllers

import (
	"net/http"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/ddollar/logger"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/convox/kernel/models"
)

var (
	log = logger.New("ns=kernel cn=build")
)

func init() {
}

func BuildCreate(rw http.ResponseWriter, r *http.Request) {
	log = log.At("create").Start()
	app := mux.Vars(r)["app"]
	repo := GetForm(r, "repo")

	build := &models.Build{App: app, Status: "building"}

	err := build.Save()

	if err != nil {
		log.Error(err)
		RenderError(rw, err)
		return
	}

	go build.Execute(repo)

	RenderText(rw, "ok")
}
