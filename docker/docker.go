package docker

import (
	"fmt"
	"os"

	docker "github.com/fsouza/go-dockerclient"
)

func PortsUsed() ([]int64, error) {
	endpoint := os.Getenv("DOCKER_HOST")
	path := os.Getenv("DOCKER_CERT_PATH")
	ca := fmt.Sprintf("%s/ca.pem", path)
	cert := fmt.Sprintf("%s/cert.pem", path)
	key := fmt.Sprintf("%s/key.pem", path)

	client, _ := docker.NewTLSClient(endpoint, cert, key, ca)

	ports := make([]int64, 0)

	cs, err := client.ListContainers(docker.ListContainersOptions{})

	for _, c := range cs {
		for _, p := range c.Ports {
			ports = append(ports, p.PublicPort)
		}
	}

	if err != nil {
		return ports, err
	}

	return ports, nil
}
