package controllers

import (
	"net/http"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/api/models"
)

func ParametersList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	app := mux.Vars(r)["app"]

	a, err := models.GetApp(app)

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

	a, err := models.GetApp(app)

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

	err = a.UpdateParams(params)

	if err != nil {
		return httperr.Server(err)
	}

	return RenderSuccess(rw)

	// vars := mux.Vars(r)
	// app := vars["app"]
	// process := vars["process"]
	// count := GetForm(r, "count")
	// memory := GetForm(r, "memory")

	// _, err := models.GetApp(app)

	// if awsError(err) == "ValidationError" {
	//   return httperr.Errorf(404, "no such app: %s", app)
	// }

	// err = models.SetParameters(app, process, count, memory)

	// if ae, ok := err.(awserr.Error); ok {
	//   if ae.Code() == "ValidationError" {
	//     switch {
	//     case strings.Index(ae.Error(), "No updates are to be performed") > -1:
	//       return httperr.Errorf(403, "no updates are to be performed: %s", app)
	//     case strings.Index(ae.Error(), "can not be updated") > -1:
	//       return httperr.Errorf(403, "app is already updating: %s", app)
	//     }
	//   }
	// }

	// if err != nil {
	//   return httperr.Server(err)
	// }

	// return RenderSuccess(rw)
}
