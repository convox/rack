package local

import (
	"context"
	"fmt"
	"os"

	"github.com/convox/logger"
	"github.com/convox/rack/pkg/manifest"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/pkg/templater"
	"github.com/convox/rack/provider/k8s"
	"github.com/gobuffalo/packr"
	am "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	DNS      string
	Platform string

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
		Platform: os.Getenv("PLATFORM"),
		Provider: kp,
		logger:   logger.Discard,
	}

	if _, err := rest.InClusterConfig(); err == nil {
		p.logger = logger.New("ns=local")
	}

	p.templater = templater.New(packr.NewBox("../local/template"), p.templateHelpers())

	kp.Engine = p

	return p, nil
}

func (p *Provider) Initialize(opts structs.ProviderOptions) error {
	log := p.logger.At("Initialize")

	if err := p.Provider.Initialize(opts); err != nil {
		return log.Error(err)
	}

	if _, err := rest.InClusterConfig(); err == nil {
		if err := p.initializePlatform(); err != nil {
			return log.Error(err)
		}

		go p.Workers()
	}

	return log.Success()
}

func (p *Provider) WithContext(ctx context.Context) structs.Provider {
	pp := *p
	pp.Provider = pp.Provider.WithContext(ctx).(*k8s.Provider)
	return &pp
}

func (p *Provider) initializePlatform() error {
	if err := p.initializePlatformDNSPort(); err != nil {
		return err
	}

	if err := p.initializePlatformDockerSocket(); err != nil {
		return err
	}

	return nil
}

func (p *Provider) initializePlatformDNSPort() error {
	if p.Cluster == nil {
		return nil
	}

	s, err := p.Cluster.CoreV1().Services("convox-system").Get("resolver", am.GetOptions{})
	if err != nil {
		return err
	}

	if len(s.Spec.Ports) != 1 {
		return fmt.Errorf("could not find resolver port")
	}

	p.DNS = fmt.Sprintf("%d", s.Spec.Ports[0].Port)

	return nil
}

func (p *Provider) initializePlatformDockerSocket() error {
	d, err := p.Cluster.ExtensionsV1beta1().Deployments(p.Rack).Get("api", am.GetOptions{})
	if err != nil {
		return err
	}

	for _, v := range d.Spec.Template.Spec.Volumes {
		if v.Name == "docker" && v.VolumeSource.HostPath != nil {
			p.Socket = v.VolumeSource.HostPath.Path
			break
		}
	}

	return nil
}
