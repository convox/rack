package controllers

import (
	"io/ioutil"
	"net/http"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/api/provider"
	"github.com/convox/rack/api/structs"
)

func EnvironmentList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	app := mux.Vars(r)["app"]

	env, err := provider.EnvironmentGet(app)

	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", app)
	}

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, env)
}

func EnvironmentSet(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)

	app := vars["app"]

	_, err := provider.EnvironmentGet(app)

	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", app)
	}

	body, err := ioutil.ReadAll(r.Body)

	if err != nil {
		return httperr.Server(err)
	}

	releaseId, err := provider.EnvironmentSet(app, structs.LoadEnvironment(body))

	if err != nil {
		return httperr.Server(err)
	}

	rw.Header().Set("Release-Id", releaseId)

	env, err := provider.EnvironmentGet(app)

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, env)
}

func EnvironmentDelete(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)
	app := vars["app"]
	name := vars["name"]

	env, err := provider.EnvironmentGet(app)

	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", app)
	}

	if err != nil {
		return httperr.Server(err)
	}

	delete(env, name)

	releaseId, err := provider.EnvironmentSet(app, env)

	if err != nil {
		return httperr.Server(err)
	}

	rw.Header().Set("Release-Id", releaseId)

	env, err = provider.EnvironmentGet(app)

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, env)
}
