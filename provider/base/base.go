package base

import (
	"context"

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

func (p *Provider) WithContext(ctx context.Context) structs.Provider {
	return p
}
