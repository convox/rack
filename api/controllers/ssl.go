package controllers

import (
	"net/http"
	"regexp"
	"strings"

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

func SSLCreate(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	a := mux.Vars(r)["app"]
	port := GetForm(r, "port")
	body := GetForm(r, "body")
	key := GetForm(r, "key")

	var validPorts = regexp.MustCompile(`[0-9]+:[0-9]+`)

	if !validPorts.MatchString(port) {
		return httperr.Errorf(403, "port must be in 443:4443 format")
	}

	ports := strings.Split(port, ":")

	ssl, err := models.CreateSSL(a, ports[0], ports[1], body, key)

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

	ssl, err := models.DeleteSSL(a)

	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", a)
	}

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, ssl)
}
