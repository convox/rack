package k8s

import (
	"bytes"
	"flag"
	"os"
	"os/exec"

	"github.com/convox/logger"
	"github.com/convox/rack/pkg/helpers"
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
	RepositoryAuth(app string) (string, string, error)
	RepositoryHost(app string) (string, bool, error)
	ResourceRender(app string, r manifest.Resource) ([]byte, error)
	Resolver() (string, error)
	ServiceHost(app string, s manifest.Service) string
	SystemHost() string
}

type Provider struct {
	Config   *rest.Config
	Cluster  kubernetes.Interface
	ID       string
	Image    string
	Engine   Engine
	Metrics  metrics.Interface
	Password string
	Provider string
	Rack     string
	Socket   string
	Storage  string
	Version  string

	logger    *logger.Logger
	templater *templater.Templater
}

func FromEnv() (*Provider, error) {
	// hack to make glog stop complaining about flag parsing
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	_ = fs.Parse([]string{})
	flag.CommandLine = fs
	runtime.ErrorHandlers = []func(error){}

	p := &Provider{
		ID:       os.Getenv("ID"),
		Image:    os.Getenv("IMAGE"),
		Password: os.Getenv("PASSWORD"),
		Provider: os.Getenv("PROVIDER"),
		Rack:     os.Getenv("RACK"),
		Socket:   helpers.CoalesceString(os.Getenv("SOCKET"), "/var/run/docker.sock"),
		Storage:  os.Getenv("STORAGE"),
		Version:  os.Getenv("VERSION"),
		logger:   logger.Discard,
	}

	if cfg, err := rest.InClusterConfig(); err == nil {
		p.Config = cfg

		kc, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			return nil, err
		}

		mc, err := metrics.NewForConfig(cfg)
		if err != nil {
			return nil, err
		}

		p.Cluster = kc
		p.Metrics = mc

		p.logger = logger.New("ns=k8s")
	}

	if p.ID == "" {
		p.ID, _ = dockerSystemId()
	}

	p.templater = templater.New(packr.NewBox("template"), p.templateHelpers())

	return p, nil
}

func (p *Provider) Apply(data []byte, prune string) ([]byte, error) {
	cmd := exec.Command("kubectl", "apply", "--prune", "-l", prune, "-f", "-")

	cmd.Stdin = bytes.NewReader(data)

	out, err := cmd.CombinedOutput()
	// fmt.Printf("output ------\n%s\n-------------\n", string(out))
	if err != nil {
		return out, err
	}

	return out, nil
}

func (p *Provider) Initialize(opts structs.ProviderOptions) error {
	log := p.logger.At("Initialize")

	if err := p.systemUpdate(p.Version); err != nil {
		return log.Error(err)
	}

	pc, err := NewPodController(p)
	if err != nil {
		return log.Error(err)
	}

	go pc.Run()

	return log.Success()
}
