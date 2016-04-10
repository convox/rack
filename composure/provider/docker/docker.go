package docker

import (
	"fmt"

	"github.com/convox/rack/composure/structs"
)

type DockerProvider struct {
	Host string
}

func NewProvider() (*DockerProvider, error) {
	p := &DockerProvider{
		Host: "tcp://192.168.99.100:2376",
	}

	return p, nil
}

func (p *DockerProvider) Load(path string) (*structs.Manifest, error) {
	return &structs.Manifest{}, fmt.Errorf("can not load")
}

func (p *DockerProvider) Pull(m *structs.Manifest) error {
	return fmt.Errorf("can not pull")
}
