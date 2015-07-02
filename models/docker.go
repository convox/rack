package models

import (
	"fmt"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/fsouza/go-dockerclient"
)

func Docker(ip string) *docker.Client {
	client, _ := docker.NewClient(fmt.Sprintf("http://%s:2376", ip))
	return client
}
