package aws

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	docker "github.com/fsouza/go-dockerclient"
)

func (p *Provider) docker(host string) (*docker.Client, error) {
	if p.DockerTLS == nil {
		return docker.NewClient(host)
	}

	// setting ca to nil, allows insecure tls verify for server certificate
	// since we are generating the certificate without knowing the server ips
	return docker.NewTLSClientFromBytes(host, p.DockerTLS.Cert, p.DockerTLS.Key, nil)
}

func (p *Provider) dockerInstance(id string) (*docker.Client, error) {
	i, err := p.describeInstance(id)
	if err != nil {
		return nil, err
	}

	host := ""

	switch {
	case p.IsTest():
		host = os.Getenv("DOCKER_HOST")
	case p.Development:
		if i.PublicIpAddress == nil {
			return nil, fmt.Errorf("can not start development builds on a private rack")
		}
		host = fmt.Sprintf("tcp://%s:2376", *i.PublicIpAddress)
	default:
		host = fmt.Sprintf("tcp://%s:2376", *i.PrivateIpAddress)
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
		TLSClientConfig:       dc.TLSConfig,
	}

	return dc, nil
}
