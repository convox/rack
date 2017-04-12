package aws

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

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

	dc, err := p.docker(host)
	if err != nil {
		return nil, err
	}

	// DefaultTransport without the proxy
	dc.HTTPClient.Transport = &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return dc, nil
}
