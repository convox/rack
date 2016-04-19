package controllers

import (
	"net/http"

	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/api/models"
	"github.com/convox/rack/api/provider"
	"github.com/gorilla/mux"
)

func LinkCreate(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	service := mux.Vars(r)["service"]

	s, err := models.GetService(service)
	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such service: %s", service)
	}
	if err != nil {
		return httperr.Server(err)
	}

	if s.Status != "running" {
		return httperr.Errorf(403, "can not link service with status: %s", s.Status)
	}

	// new services should use the provider interfaces
	if s.Type == "syslog" {
		s, err := provider.ServiceLink(service, GetForm(r, "app"), GetForm(r, "process"))
		if err != nil {
			return httperr.Server(err)
		}

		return RenderJson(rw, s)
	}

	if s.Type != "papertrail" {
		return httperr.Errorf(403, "linking is not yet implemented for service type: %s", s.Type)
	}

	app := GetForm(r, "app")

	a, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", app)
	}

	if err != nil {
		return httperr.Server(err)
	}

	err = s.LinkPapertrail(*a)

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, s)
}

func LinkDelete(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	app := mux.Vars(r)["app"]
	service := mux.Vars(r)["service"]

	s, err := models.GetService(service)
	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such service: %s", service)
	}
	if err != nil {
		return httperr.Server(err)
	}

	if s.Status != "running" {
		return httperr.Errorf(403, "can not unlink service with status: %s", s.Status)
	}

	// new services should use the provider interfaces
	if s.Type == "syslog" {
		s, err := provider.ServiceUnlink(service, app, GetForm(r, "process"))
		if err != nil {
			return httperr.Server(err)
		}

		return RenderJson(rw, s)
	}

	if s.Type != "papertrail" {
		return httperr.Errorf(403, "unlinking is not yet implemented for service type: %s", s.Type)
	}

	a, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", app)
	}

	if err != nil {
		return httperr.Server(err)
	}

	err = s.UnlinkPapertrail(*a)

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, s)
}
