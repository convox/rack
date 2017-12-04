package controllers

import (
	"net/http"

	"github.com/convox/rack/api/httperr"
)

func RegistryList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	registries, err := Provider.RegistryList()
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, registries)
}

func RegistryCreate(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	server := GetForm(r, "server")
	username := GetForm(r, "username")
	password := GetForm(r, "password")

	registry, err := Provider.RegistryAdd(server, username, password)
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, registry)
}

func RegistryDelete(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	server := r.FormValue("server")

	if err := Provider.RegistryDelete(server); err != nil {
		return httperr.Server(err)
	}

	return RenderSuccess(rw)
}
