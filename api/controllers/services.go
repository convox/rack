package controllers

import (
	"net/http"
	"strings"

	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/api/models"
	"github.com/gorilla/mux"
)

func ServiceList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	services, err := models.Provider().ServiceList()

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, services)
}

func ServiceShow(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	service := mux.Vars(r)["service"]

	sv, err := models.Provider().ServiceGet(service)
	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such service: %s", service)
	}
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, sv)
}

func ServiceCreate(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	params, err := formHash(r)
	if err != nil {
		return httperr.Server(err)
	}

	delete(params, "name")
	delete(params, "type")

	s, err := models.Provider().ServiceCreate(r.Form.Get("name"), r.Form.Get("type"), params)
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, s)
}

func ServiceDelete(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	service := mux.Vars(r)["service"]

	s, err := models.Provider().ServiceGet(service)
	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such service: %s", service)
	}
	if err != nil {
		return httperr.Server(err)
	}

	s, err = models.Provider().ServiceDelete(service)
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, s)
}

func ServiceUpdate(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	service := mux.Vars(r)["service"]

	params, err := formHash(r)
	if err != nil {
		return httperr.Server(err)
	}

	s, err := models.Provider().ServiceUpdate(service, params)
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, s)
}

func convoxifyCloudformationError(msg string) string {
	newMsg := strings.Replace(msg, "do not exist in the template", "are not supported by this service", 1)
	newMsg = strings.Replace(newMsg, "Parameters:", "Options:", 1)
	return newMsg
}
