package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/fsouza/go-dockerclient"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/api/models"
)

func getEnvAuthConfigurations(env models.Environment) (docker.AuthConfigurations119, error) {
	ac := docker.AuthConfigurations119{}

	data := []byte(env["DOCKER_AUTH_DATA"])

	if len(data) > 0 {
		if err := json.Unmarshal(data, &ac); err != nil {
			return ac, err
		}
	}

	return ac, nil
}

func RegistryList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	env, err := models.GetRackEnvironment()

	if err != nil {
		return httperr.Server(err)
	}

	acs, err := getEnvAuthConfigurations(env)

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

	env, err := models.GetRackEnvironment()

	if err != nil {
		return httperr.Server(err)
	}

	acs, err := getEnvAuthConfigurations(env)

	if err != nil {
		return httperr.Server(err)
	}

	acs[ac.ServerAddress] = ac

	dat, err := json.Marshal(acs)

	if err != nil {
		return httperr.Server(err)
	}

	env["DOCKER_AUTH_DATA"] = string(dat)

	err = models.PutRackEnvironment(env)

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, ac)
}

func RegistryDelete(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	registry := mux.Vars(r)["registry"]

	env, err := models.GetRackEnvironment()

	if err != nil {
		return httperr.Server(err)
	}

	acs, err := getEnvAuthConfigurations(env)

	if err != nil {
		return httperr.Server(err)
	}

	ac, ok := acs[registry]

	if !ok {
		return httperr.Errorf(404, "no such registry: %s", registry)
	}

	delete(acs, registry)

	dat, err := json.Marshal(acs)

	if err != nil {
		return httperr.Server(err)
	}

	env["DOCKER_AUTH_DATA"] = string(dat)

	err = models.PutRackEnvironment(env)

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, ac)
}
