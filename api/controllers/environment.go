package controllers

import (
	"io/ioutil"
	"net/http"

	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/structs"
	"github.com/gorilla/mux"
)

func EnvironmentGet(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	app := mux.Vars(r)["app"]

	env, err := Provider.EnvironmentGet(app)
	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", app)
	}
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, env)
}

func EnvironmentSet(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	app := mux.Vars(r)["app"]

	_, err := Provider.EnvironmentGet(app)
	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", app)
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return httperr.Server(err)
	}

	env := structs.Environment{}

	if err := env.Load(body); err != nil {
		return httperr.Server(err)
	}

	release, err := Provider.EnvironmentPut(app, env)
	if err != nil {
		return httperr.Server(err)
	}

	rw.Header().Set("Release-Id", release)

	env, err = Provider.EnvironmentGet(app)
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, env)
}

func EnvironmentDelete(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)
	app := vars["app"]
	name := vars["name"]

	env, err := Provider.EnvironmentGet(app)
	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", app)
	}
	if err != nil {
		return httperr.Server(err)
	}

	delete(env, name)

	release, err := Provider.EnvironmentPut(app, env)
	if err != nil {
		return httperr.Server(err)
	}

	rw.Header().Set("Release-Id", release)

	env, err = Provider.EnvironmentGet(app)
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, env)
}
