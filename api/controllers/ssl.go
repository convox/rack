package controllers

import (
	"net/http"
	"strconv"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/api/models"
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

func SSLCreate(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	a := mux.Vars(r)["app"]
	process := GetForm(r, "process")
	port := GetForm(r, "port")
	body := GetForm(r, "body")
	key := GetForm(r, "key")

	if process == "" {
		return httperr.Errorf(403, "must specify a process")
	}

	portn, err := strconv.Atoi(port)

	if err != nil {
		return httperr.Errorf(403, "port must be numeric")
	}

	ssl, err := models.CreateSSL(a, process, portn, body, key)

	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "%s", err)
	}

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, ssl)
}

func SSLDelete(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)
	app := vars["app"]
	process := vars["process"]
	port := vars["port"]

	if process == "" {
		return httperr.Errorf(403, "must specify a process")
	}

	portn, err := strconv.Atoi(port)

	if err != nil {
		return httperr.Errorf(403, "port must be numeric")
	}

	ssl, err := models.DeleteSSL(app, process, portn)

	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", app)
	}

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, ssl)
}
