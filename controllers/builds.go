package controllers

import (
	"net/http"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/convox/kernel/models"
)

func init() {
}

func BuildCreate(rw http.ResponseWriter, r *http.Request) {
	app := mux.Vars(r)["app"]
	repo := GetForm(r, "repo")

	build := &models.Build{App: app, Status: "building"}

	err := build.Save()

	if err != nil {
		RenderError(rw, err)
		return
	}

	go build.Execute(repo)

	RenderText(rw, "ok")
}
