package base

import (
	"github.com/convox/rack/pkg/structs"
)

type Provider struct {
}

func FromEnv() (*Provider, error) {
	p := &Provider{}

	return p, nil
}

func (p *Provider) Initialize(opts structs.ProviderOptions) error {
	return nil
}
