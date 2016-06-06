package controllers

import (
	"net/http"

	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/api/provider"
	"github.com/gorilla/mux"
)

func LinkCreate(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	service := mux.Vars(r)["service"]

	s, err := provider.ServiceGet(service)
	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such service: %s", service)
	}
	if err != nil {
		return httperr.Server(err)
	}
	if s.Status != "running" {
		return httperr.Errorf(403, "can not link service with status: %s", s.Status)
	}

	s, err = provider.ServiceLink(service, GetForm(r, "app"), GetForm(r, "process"))
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, s)
}

func LinkDelete(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	app := mux.Vars(r)["app"]
	service := mux.Vars(r)["service"]

	s, err := provider.ServiceGet(service)
	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such service: %s", service)
	}
	if err != nil {
		return httperr.Server(err)
	}
	if s.Status != "running" {
		return httperr.Errorf(403, "can not unlink service with status: %s", s.Status)
	}

	s, err = provider.ServiceUnlink(service, app, GetForm(r, "process"))
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, s)
}
