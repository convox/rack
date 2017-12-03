package controllers

import (
	"net/http"
	"os"

	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/structs"
	"github.com/gorilla/mux"
)

func ParametersList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	app := mux.Vars(r)["app"]

	if app == os.Getenv("RACK") {
		s, err := Provider.SystemGet()
		if err != nil {
			return httperr.Server(err)
		}

		return RenderJson(rw, s.Parameters)
	}

	a, err := Provider.AppGet(app)
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

	r.ParseMultipartForm(2048)

	params := map[string]string{}

	for key, values := range r.Form {
		params[key] = values[0]
	}

	if app == os.Getenv("RACK") {
		if err := Provider.SystemUpdate(structs.SystemUpdateOptions{Parameters: params}); err != nil {
			return httperr.Server(err)
		}
		return RenderSuccess(rw)
	}

	_, err := Provider.AppGet(app)
	if err != nil {
		if awsError(err) == "ValidationError" {
			return httperr.Errorf(404, "no such app: %s", app)
		}
		return httperr.Server(err)
	}

	if err := Provider.AppUpdate(app, structs.AppUpdateOptions{Parameters: params}); err != nil {
		return httperr.Server(err)
	}

	return RenderSuccess(rw)
}
