package controllers

import (
	"net/http"
	"strings"

	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/structs"
	"github.com/gorilla/mux"
)

// ResourceList lists resources.
func ResourceList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	resources, err := Provider.ResourceList()

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, resources)
}

// ResourceShow shows a resource.
func ResourceShow(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	resource := mux.Vars(r)["resource"]

	sv, err := Provider.ResourceGet(resource)
	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such resource: %s", resource)
	}
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, sv)
}

// ResourceCreate creates a resource.
func ResourceCreate(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	params, err := formHash(r)
	if err != nil {
		return httperr.Server(err)
	}

	delete(params, "name")
	delete(params, "type")

	s, err := Provider.ResourceCreate(r.Form.Get("name"), r.Form.Get("type"), structs.ResourceCreateOptions{Parameters: params})
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, s)
}

// ResourceDelete deletes a resource.
func ResourceDelete(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	resource := mux.Vars(r)["resource"]

	s, err := Provider.ResourceGet(resource)
	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such resource: %s", resource)
	}
	if err != nil {
		return httperr.Server(err)
	}

	s, err = Provider.ResourceDelete(resource)
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, s)
}

// ResourceUpdate updates a resource.
func ResourceUpdate(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	resource := mux.Vars(r)["resource"]

	params, err := formHash(r)
	if err != nil {
		return httperr.Server(err)
	}

	s, err := Provider.ResourceUpdate(resource, params)
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
