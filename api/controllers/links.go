package controllers

import (
	"net/http"

	"github.com/convox/rack/api/httperr"
	"github.com/gorilla/mux"
)

func LinkCreate(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	resource := mux.Vars(r)["resource"]

	s, err := Provider.ResourceGet(resource)
	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such resource: %s", resource)
	}
	if err != nil {
		return httperr.Server(err)
	}
	if s.Status != "running" {
		return httperr.Errorf(403, "can not link resource with status: %s", s.Status)
	}

	s, err = Provider.ResourceLink(resource, GetForm(r, "app"), GetForm(r, "process"))
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, s)
}

func LinkDelete(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	app := mux.Vars(r)["app"]
	resource := mux.Vars(r)["resource"]

	s, err := Provider.ResourceGet(resource)
	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such resource: %s", resource)
	}
	if err != nil {
		return httperr.Server(err)
	}
	if s.Status != "running" {
		return httperr.Errorf(403, "can not unlink resource with status: %s", s.Status)
	}

	s, err = Provider.ResourceUnlink(resource, app, GetForm(r, "process"))
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, s)
}
