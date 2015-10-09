package controllers

import (
	"net/http"

	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/api/models"
	"github.com/gorilla/mux"
)

func SSLList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	a := mux.Vars(r)["app"]

	ssls, err := models.ListSSLs(a)

	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", a)
	}

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, ssls)
}
