package controllers

import (
	"net/http"

	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/api/models"
)

func RegistryList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	registries, err := models.Provider().RegistryList()
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, registries)
}

func RegistryCreate(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	server := GetForm(r, "server")
	username := GetForm(r, "username")
	password := GetForm(r, "password")

	registry, err := models.Provider().RegistryAdd(server, username, password)
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, registry)
}

func RegistryDelete(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	server := r.FormValue("server")

	if err := models.Provider().RegistryDelete(server); err != nil {
		return httperr.Server(err)
	}

	return RenderSuccess(rw)
}
