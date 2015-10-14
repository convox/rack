package controllers

import (
	"net/http"
	"regexp"

	"github.com/convox/rack/api/Godeps/_workspace/src/github.com/gorilla/mux"
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
	port := GetForm(r, "port")
	body := GetForm(r, "body")
	key := GetForm(r, "key")

	var validPorts = regexp.MustCompile(`[0-9]+`)

	if !validPorts.MatchString(port) {
		return httperr.Errorf(403, "balancer port must be in 443 format")
	}

	ssl, err := models.CreateSSL(a, port, body, key)

	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "%s", err)
	}

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, ssl)
}

func SSLDelete(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	a := mux.Vars(r)["app"]
	port := mux.Vars(r)["port"]

	ssl, err := models.DeleteSSL(a, port)

	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", a)
	}

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, ssl)
}
