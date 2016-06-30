package controllers

import (
	"io/ioutil"
	"net/http"

	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/api/models"
	"github.com/gorilla/mux"
)

func EnvironmentList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	app := mux.Vars(r)["app"]

	env, err := models.GetEnvironment(app)

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

	_, err := models.GetEnvironment(app)
	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", app)
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return httperr.Server(err)
	}

	releaseId, err := models.PutEnvironment(app, models.LoadEnvironment(body))
	if err != nil {
		return httperr.Server(err)
	}

	rw.Header().Set("Release-Id", releaseId)

	env, err := models.GetEnvironment(app)
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, env)
}

func EnvironmentDelete(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)
	app := vars["app"]
	name := vars["name"]

	env, err := models.GetEnvironment(app)

	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", app)
	}

	if err != nil {
		return httperr.Server(err)
	}

	delete(env, name)

	releaseId, err := models.PutEnvironment(app, env)

	if err != nil {
		return httperr.Server(err)
	}

	rw.Header().Set("Release-Id", releaseId)

	env, err = models.GetEnvironment(app)

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, env)
}
