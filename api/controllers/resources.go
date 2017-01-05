package controllers

import (
	"net/http"
	"strings"

	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/api/models"
	"github.com/gorilla/mux"
)

func ResourceList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	resources, err := models.Provider().ResourceList()

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, resources)
}

func ResourceShow(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	resource := mux.Vars(r)["resource"]

	sv, err := models.Provider().ResourceGet(resource)
	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such resource: %s", resource)
	}
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, sv)
}

func ResourceCreate(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	params, err := formHash(r)
	if err != nil {
		return httperr.Server(err)
	}

	delete(params, "name")
	delete(params, "type")

	s, err := models.Provider().ResourceCreate(r.Form.Get("name"), r.Form.Get("type"), params)
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, s)
}

func ResourceDelete(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	resource := mux.Vars(r)["resource"]

	s, err := models.Provider().ResourceGet(resource)
	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such resource: %s", resource)
	}
	if err != nil {
		return httperr.Server(err)
	}

	s, err = models.Provider().ResourceDelete(resource)
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, s)
}

func ResourceUpdate(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	resource := mux.Vars(r)["resource"]

	params, err := formHash(r)
	if err != nil {
		return httperr.Server(err)
	}

	s, err := models.Provider().ResourceUpdate(resource, params)
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, s)
}

func convoxifyCloudformationError(msg string) string {
	newMsg := strings.Replace(msg, "do not exist in the template", "are not supported by this resource", 1)
	newMsg = strings.Replace(newMsg, "Parameters:", "Options:", 1)
	return newMsg
}
