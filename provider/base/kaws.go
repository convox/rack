package base

import (
	"os"

	"github.com/convox/rack/pkg/structs"
)

type Provider struct {
	Region string
}

func FromEnv() (*Provider, error) {
	p := &Provider{
		Region: os.Getenv("AWS_REGION"),
	}

	return p, nil
}

func (p *Provider) Initialize(opts structs.ProviderOptions) error {
	return nil
}
