package local

import (
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/pkg/templater"
	"github.com/convox/rack/provider/k8s"
	"github.com/gobuffalo/packr"
	"k8s.io/client-go/rest"
)

// var (
//   Templater *templater.Templater
// )

// func init() {
//   Templater = templater.New(packr.NewBox("template"), templateHelpers())
// }

type Provider struct {
	*k8s.Provider

	templater *templater.Templater
}

func FromEnv() (*Provider, error) {
	kp, err := k8s.FromEnv()
	if err != nil {
		return nil, err
	}

	p := &Provider{
		Provider: kp,
	}

	p.templater = templater.New(packr.NewBox("template"), p.templateHelpers())

	kp.Engine = p

	return p, nil
}

func (p *Provider) Initialize(opts structs.ProviderOptions) error {
	if err := p.Provider.Initialize(opts); err != nil {
		return err
	}

	if _, err := rest.InClusterConfig(); err == nil {
		go p.Workers()
	}

	return nil
}
