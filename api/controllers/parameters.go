package controllers

import (
	"net/http"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/api/models"
	"github.com/convox/rack/api/provider"
)

func ParametersList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	app := mux.Vars(r)["app"]

	a, err := provider.AppGet(app)

	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", app)
	}

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, a.Parameters)
}

func ParametersSet(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	app := mux.Vars(r)["app"]

	a, err := provider.AppGet(app)

	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", app)
	}

	if err != nil {
		return httperr.Server(err)
	}

	r.ParseMultipartForm(2048)

	params := map[string]string{}

	for key, values := range r.Form {
		params[key] = values[0]
	}

	err = models.AppUpdateParams(a, params)

	if err != nil {
		return httperr.Server(err)
	}

	return RenderSuccess(rw)
}
