package aws

import (
	"fmt"
	"os"

	"github.com/fsouza/go-dockerclient"
)

func (p *AWSProvider) docker(host string) (*docker.Client, error) {
	return docker.NewClient(host)
}

func (p *AWSProvider) dockerInstance(id string) (*docker.Client, error) {
	i, err := p.describeInstance(id)
	if err != nil {
		return nil, err
	}

	host := ""

	switch {
	case p.IsTest():
		host = fmt.Sprintf("http://%s", os.Getenv("DOCKER_HOST"))
	case p.Development:
		host = fmt.Sprintf("http://%s:2376", *i.PublicIpAddress)
	default:
		host = fmt.Sprintf("http://%s:2376", *i.PrivateIpAddress)
	}

	return p.docker(host)
}
