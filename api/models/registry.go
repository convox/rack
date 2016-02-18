package models

import (
	"encoding/json"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/fsouza/go-dockerclient"
)

func GetPrivateRegistriesAuth() (Environment, docker.AuthConfigurations119, error) {
	acs := docker.AuthConfigurations119{}

	env, err := GetRackSettings()

	if err != nil {
		return env, acs, err
	}

	data := []byte(env["DOCKER_AUTH_DATA"])

	if len(data) > 0 {
		if err := json.Unmarshal(data, &acs); err != nil {
			return env, acs, err
		}
	}

	return env, acs, nil
}

func LoginPrivateRegistries() error {
	_, acs, err := GetPrivateRegistriesAuth()

	if err != nil {
		return err
	}

	for _, ac := range acs {
		DockerLogin(ac)
	}

	return nil
}
