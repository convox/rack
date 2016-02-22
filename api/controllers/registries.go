package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/fsouza/go-dockerclient"

	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/api/models"
)

func RegistryList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	_, acs, err := models.GetPrivateRegistriesAuth()

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, acs)
}

func RegistryCreate(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	ac := docker.AuthConfiguration{
		Username:      GetForm(r, "username"),
		Password:      GetForm(r, "password"),
		Email:         GetForm(r, "email"),
		ServerAddress: GetForm(r, "serveraddress"),
	}

	_, err := models.DockerLogin(ac)

	if err != nil {
		return httperr.Errorf(400, "Could not login to server with provided credentials")
	}

	env, acs, err := models.GetPrivateRegistriesAuth()

	if err != nil {
		return httperr.Server(err)
	}

	acs[ac.ServerAddress] = ac

	dat, err := json.Marshal(acs)

	if err != nil {
		return httperr.Server(err)
	}

	env["DOCKER_AUTH_DATA"] = string(dat)

	err = models.PutRackSettings(env)

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, ac)
}

func RegistryDelete(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	// server := mux.Vars(r)["server"]
	server := r.FormValue("server")

	env, acs, err := models.GetPrivateRegistriesAuth()

	if err != nil {
		return httperr.Server(err)
	}

	ac, ok := acs[server]

	if !ok {
		return httperr.Errorf(404, "no such registry: %s", server)
	}

	models.DockerLogout(ac)
	delete(acs, server)

	dat, err := json.Marshal(acs)

	if err != nil {
		return httperr.Server(err)
	}

	env["DOCKER_AUTH_DATA"] = string(dat)

	err = models.PutRackSettings(env)

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, ac)
}
