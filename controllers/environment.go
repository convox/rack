package controllers

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/convox/kernel/helpers"
	"github.com/convox/kernel/models"
)

func EnvironmentSet(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	app := vars["app"]

	body, err := ioutil.ReadAll(r.Body)

	if err != nil {
		helpers.Error(nil, err)
		RenderError(rw, err)
		return
	}

	err = models.PutEnvironment(app, models.LoadEnvironment(body))

	if err != nil {
		helpers.Error(nil, err)
		RenderError(rw, err)
		return
	}

	RenderText(rw, "ok")
}

func EnvironmentCreate(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	app := vars["app"]
	name := vars["name"]
	value := GetForm(r, "value")

	env, err := models.GetEnvironment(app)

	if err != nil {
		helpers.Error(nil, err)
		RenderError(rw, err)
		return
	}

	env[strings.ToUpper(name)] = value

	err = models.PutEnvironment(app, env)

	if err != nil {
		helpers.Error(nil, err)
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

	if awsError(err) == "ValidationError" {
		RenderNotFound(rw, fmt.Sprintf("no such app: %s", app))
		return
	}

	if err != nil {
		helpers.Error(nil, err)
		RenderError(rw, err)
		return
	}

	delete(env, name)

	err = models.PutEnvironment(app, env)

	if err != nil {
		helpers.Error(nil, err)
		RenderError(rw, err)
		return
	}

	RenderText(rw, "ok")
}
