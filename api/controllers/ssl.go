package controllers

import (
	"net/http"
	"strconv"
	"strings"

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
	arn := GetForm(r, "arn")
	chain := GetForm(r, "chain")
	body := GetForm(r, "body")
	key := GetForm(r, "key")
	secure := GetForm(r, "secure")

	if process == "" {
		return httperr.Errorf(403, "must specify a process")
	}

	portn, err := strconv.Atoi(port)

	if err != nil {
		return httperr.Errorf(403, "port must be numeric")
	}

	if (arn != "") && !validateARNFormat(arn) {
		return httperr.Errorf(403, "arn must follow the AWS ARN format")
	}

	ssl, err := models.CreateSSL(a, process, portn, arn, body, key, chain, (secure == "true"))

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

func SSLUpdate(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	a := mux.Vars(r)["app"]
	process := GetForm(r, "process")
	port := GetForm(r, "port")
	arn := GetForm(r, "arn")
	chain := GetForm(r, "chain")
	body := GetForm(r, "body")
	key := GetForm(r, "key")

	if process == "" {
		return httperr.Errorf(403, "must specify a process")
	}

	portn, err := strconv.Atoi(port)

	if err != nil {
		return httperr.Errorf(403, "port must be numeric")
	}

	ssl, err := models.UpdateSSL(a, process, portn, arn, body, key, chain)

	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "%s", err)
	}

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, ssl)
}

func validateARNFormat(arn string) bool {
	return strings.HasPrefix(strings.ToLower(arn), "arn:")
}
