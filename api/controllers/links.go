package controllers

import (
	"fmt"
	"net/http"

	"github.com/convox/rack/api/models"
	"github.com/gorilla/mux"
)

func LinkCreate(rw http.ResponseWriter, r *http.Request) error {
	service := mux.Vars(r)["service"]

	s, err := models.GetService(service)

	if awsError(err) == "ValidationError" {
		return RenderNotFound(rw, fmt.Sprintf("no such service: %s", service))
	}

	if err != nil {
		return err
	}

	if s.Status != "running" {
		return RenderForbidden(rw, fmt.Sprintf("can not link service with status: %s", s.Status))
	}

	if s.Type != "papertrail" {
		return RenderForbidden(rw, fmt.Sprintf("linking is not yet implemented for service type: %s", s.Type))
	}

	app := GetForm(r, "app")

	a, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		return RenderNotFound(rw, fmt.Sprintf("no such app: %s", app))
	}

	if err != nil {
		return err
	}

	err = s.LinkPapertrail(*a)

	if err != nil {
		return err
	}

	return RenderJson(rw, s)
}

func LinkDelete(rw http.ResponseWriter, r *http.Request) error {
	app := mux.Vars(r)["app"]
	service := mux.Vars(r)["service"]

	s, err := models.GetService(service)

	if awsError(err) == "ValidationError" {
		return RenderNotFound(rw, fmt.Sprintf("no such service: %s", service))
	}

	if err != nil {
		return err
	}

	if s.Status != "running" {
		return RenderForbidden(rw, fmt.Sprintf("can not unlink service with status: %s", s.Status))
	}

	if s.Type != "papertrail" {
		return RenderForbidden(rw, fmt.Sprintf("unlinking is not yet implemented for service type: %s", s.Type))
	}

	a, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		return RenderNotFound(rw, fmt.Sprintf("no such app: %s", app))
	}

	if err != nil {
		return err
	}

	err = s.UnlinkPapertrail(*a)

	if err != nil {
		return err
	}

	return RenderJson(rw, s)
}
