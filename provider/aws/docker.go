package aws

import (
	"fmt"

	"github.com/fsouza/go-dockerclient"
)

func (p *AWSProvider) docker(host string) (*docker.Client, error) {
	return docker.NewClient(host)
}

func (p *AWSProvider) dockerInstance(id string) (*docker.Client, error) {
	host, err := p.describeInstance(id)
	if err != nil {
		return nil, err
	}

	ip := *host.PrivateIpAddress

	if p.Development {
		ip = *host.PublicIpAddress
	}

	return p.docker(fmt.Sprintf("http://%s:2376", ip))
}
