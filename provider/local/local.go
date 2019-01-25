package local

import (
	"github.com/convox/logger"
	"github.com/convox/rack/pkg/manifest"
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

	logger    *logger.Logger
	templater *templater.Templater
}

func FromEnv() (*Provider, error) {
	manifest.DefaultCpu = 64
	manifest.DefaultMem = 256

	kp, err := k8s.FromEnv()
	if err != nil {
		return nil, err
	}

	p := &Provider{
		Provider: kp,
		logger:   logger.Discard,
	}

	if _, err := rest.InClusterConfig(); err == nil {
		p.logger = logger.New("ns=local")
	}

	p.templater = templater.New(packr.NewBox("template"), p.templateHelpers())

	kp.Engine = p

	return p, nil
}

func (p *Provider) Initialize(opts structs.ProviderOptions) error {
	log := p.logger.At("Initialize")

	if err := p.systemUpdate(p.Version); err != nil {
		return log.Error(err)
	}

	if err := p.Provider.Initialize(opts); err != nil {
		return log.Error(err)
	}

	if _, err := rest.InClusterConfig(); err == nil {
		go p.Workers()
	}

	return log.Success()
}
