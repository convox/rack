package controllers

import (
	"io/ioutil"
	"net/http"

	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/helpers"
	"github.com/convox/rack/options"
	"github.com/convox/rack/structs"
	"github.com/gorilla/mux"
)

func EnvironmentGet(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	app := mux.Vars(r)["app"]

	env, err := helpers.AppEnvironment(Provider, app)
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

	_, err := helpers.AppEnvironment(Provider, app)
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

	rr, err := Provider.ReleaseCreate(app, structs.ReleaseCreateOptions{Env: options.String(env.String())})
	if err != nil {
		return httperr.Server(err)
	}

	rw.Header().Set("Release-Id", rr.Id)

	env, err = helpers.AppEnvironment(Provider, app)
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, env)
}

func EnvironmentDelete(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)
	app := vars["app"]
	name := vars["name"]

	env, err := helpers.AppEnvironment(Provider, app)
	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", app)
	}
	if err != nil {
		return httperr.Server(err)
	}

	delete(env, name)

	rr, err := Provider.ReleaseCreate(app, structs.ReleaseCreateOptions{Env: options.String(env.String())})
	if err != nil {
		return httperr.Server(err)
	}

	rw.Header().Set("Release-Id", rr.Id)

	env, err = helpers.AppEnvironment(Provider, app)
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, env)
}
