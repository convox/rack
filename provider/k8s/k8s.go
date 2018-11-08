package k8s

import (
	"flag"
	"os"

	"github.com/convox/rack/pkg/manifest"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/pkg/templater"
	"github.com/gobuffalo/packr"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	metrics "k8s.io/metrics/pkg/client/clientset_generated/clientset"
)

type Engine interface {
	AppRepository(app string) (string, bool, error)
	ResourceRender(app string, r manifest.Resource) ([]byte, error)
	ServiceHost(app, service string) string
}

type Provider struct {
	Config  *rest.Config
	Cluster kubernetes.Interface
	// HostFunc func(app, service string) string
	Image    string
	Engine   Engine
	Metrics  metrics.Interface
	Password string
	Provider string
	Rack     string
	// RepoFunc func(app string) (string, bool, error)
	Storage string
	Version string

	templater *templater.Templater
}

func init() {
	// hack to make glog stop complaining about flag parsing
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	_ = fs.Parse([]string{})
	flag.CommandLine = fs
}

func FromEnv() (*Provider, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	kc, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	mc, err := metrics.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	p := &Provider{
		Config:   cfg,
		Cluster:  kc,
		Image:    os.Getenv("IMAGE"),
		Metrics:  mc,
		Password: os.Getenv("PASSWORD"),
		Provider: os.Getenv("PROVIDER"),
		Rack:     os.Getenv("RACK"),
		Storage:  os.Getenv("STORAGE"),
		Version:  os.Getenv("VERSION"),
	}

	p.templater = templater.New(packr.NewBox("template"), p.templateHelpers())

	return p, nil
}

func (p *Provider) Initialize(opts structs.ProviderOptions) error {
	runtime.ErrorHandlers = []func(error){}

	pc, err := NewPodController(p)
	if err != nil {
		return err
	}

	go pc.Run()

	return nil
}
