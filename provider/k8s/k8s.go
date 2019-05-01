package k8s

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/convox/logger"
	"github.com/convox/rack/pkg/atom"
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
	AppIdles(app string) (bool, error)
	AppStatus(app string) (string, error)
	Log(app, stream string, ts time.Time, message string) error
	ReleasePromote(app, id string, opts structs.ReleasePromoteOptions) error
	RepositoryAuth(app string) (string, string, error)
	RepositoryHost(app string) (string, bool, error)
	ResourceRender(app string, r manifest.Resource) ([]byte, error)
	Resolver() (string, error)
	ServiceHost(app string, s manifest.Service) string
	// SystemAnnotations(service string) map[string]string
	SystemHost() string
	SystemStatus() (string, error)
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

	atom      *atom.Client
	ctx       context.Context
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
		ctx:      context.Background(),
		logger:   logger.Discard,
	}

	p.templater = templater.New(packr.NewBox("../k8s/template"), p.templateHelpers())

	if cfg, err := rest.InClusterConfig(); err == nil {
		p.logger = logger.New("ns=k8s")

		p.Config = cfg

		ac, err := atom.New(cfg)
		if err != nil {
			return nil, err
		}

		p.atom = ac

		kc, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			return nil, err
		}

		p.Cluster = kc

		mc, err := metrics.NewForConfig(cfg)
		if err != nil {
			return nil, err
		}

		p.Metrics = mc
	}

	if p.ID == "" {
		p.ID, _ = dockerSystemId()
	}

	return p, nil
}

func (p *Provider) Initialize(opts structs.ProviderOptions) error {
	log := p.logger.At("Initialize")

	if err := p.initializeAtom(); err != nil {
		return log.Error(err)
	}

	dc, err := NewDeploymentController(p)
	if err != nil {
		return log.Error(err)
	}

	ec, err := NewEventController(p)
	if err != nil {
		return log.Error(err)
	}

	nc, err := NewNodeController(p)
	if err != nil {
		return log.Error(err)
	}

	pc, err := NewPodController(p)
	if err != nil {
		return log.Error(err)
	}

	go dc.Run()
	go ec.Run()
	go nc.Run()
	go pc.Run()

	return log.Success()
}

func (p *Provider) Context() context.Context {
	return p.ctx
}

func (p *Provider) WithContext(ctx context.Context) structs.Provider {
	pp := *p
	pp.ctx = ctx
	return &pp
}

func (p *Provider) initializeAtom() error {
	params := map[string]interface{}{
		"Version": p.Version,
	}

	data, err := p.RenderTemplate("atom", params)
	if err != nil {
		return err
	}

	cmd := exec.Command("kubectl", "apply", "-f", "-")

	cmd.Stdin = bytes.NewReader(data)

	if out, err := cmd.CombinedOutput(); err != nil {
		return errors.New(strings.TrimSpace(string(out)))
	}

	return nil
}
