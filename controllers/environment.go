package controllers

import (
	"net/http"
	"strings"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/convox/kernel/models"
)

func EnvironmentCreate(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	app := vars["app"]
	name := GetForm(r, "name")
	value := GetForm(r, "value")

	env, err := models.GetEnvironment(app)

	if err != nil {
		RenderError(rw, err)
		return
	}

	env[strings.ToUpper(name)] = value

	err = models.PutEnvironment(app, env)

	if err != nil {
		RenderError(rw, err)
		return
	}

	RenderText(rw, "ok")
}

func EnvironmentDelete(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	app := vars["app"]
	name := vars["name"]

	env, err := models.GetEnvironment(app)

	if err != nil {
		RenderError(rw, err)
		return
	}

	delete(env, name)

	err = models.PutEnvironment(app, env)

	if err != nil {
		RenderError(rw, err)
		return
	}

	RenderText(rw, "ok")
}
