package local

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/boltdb/bolt"
	"github.com/convox/logger"
	"github.com/convox/rack/router"
	"github.com/convox/rack/structs"
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

type Provider struct {
	Combined bool
	Image    string
	Name     string
	Root     string
	Router   string
	Test     bool
	Version  string
	Volume   string

	ctx    context.Context
	db     *bolt.DB
	logs   *logger.Logger
	router *router.Client
}

func FromEnv() *Provider {
	return &Provider{
		Combined: os.Getenv("COMBINED") == "true",
		Image:    coalesce(os.Getenv("IMAGE"), "convox/rack"),
		Name:     coalesce(os.Getenv("NAME"), "convox"),
		Root:     coalesce(os.Getenv("PROVIDER_ROOT"), "/var/convox"),
		Router:   coalesce(os.Getenv("PROVIDER_ROUTER"), "10.42.0.0"),
		Test:     os.Getenv("TEST") == "true",
		Version:  coalesce(os.Getenv("VERSION"), "latest"),
		Volume:   coalesce(os.Getenv("PROVIDER_VOLUME"), "/var/convox"),
		logs:     logger.NewWriter("", ioutil.Discard),
	}
}

func (p *Provider) Initialize(opts structs.ProviderOptions) error {
	if opts.Logs != nil {
		p.logs = logger.NewWriter("ns=provider.local", opts.Logs)
	} else {
		p.logs = logger.New("ns=provider.local")
	}

	if err := os.MkdirAll(p.Root, 0700); err != nil {
		return err
	}

	db, err := bolt.Open(filepath.Join(p.Root, fmt.Sprintf("%s.db", p.Name)), 0600, nil)
	if err != nil {
		return err
	}

	p.db = db

	if _, err := p.createRootBucket("rack"); err != nil {
		return err
	}

	if p.Router != "none" {
		p.router = router.NewClient(coalesce(os.Getenv("PROVIDER_ROUTER"), "10.42.0.0"))

		if err := p.routerCheck(); err != nil {
			return err
		}

		if err := p.routerRegister(); err != nil {
			return err
		}
	}

	if p.Combined {
		go p.Workers()
	}

	return nil
}

func (p *Provider) logger(at string) *logger.Logger {
	if p.Test {
		return logger.NewWriter("", ioutil.Discard)
	}

	return p.logs.At(at).Start()
}

// shutdown cleans up any running resources and exit
func (p *Provider) shutdown() error {
	cs, err := containersByLabels(map[string]string{
		"convox.rack": p.Name,
	})
	if err != nil {
		return err
	}

	var wg sync.WaitGroup

	for _, c := range cs {
		wg.Add(1)
		go p.containerStopAsync(c.Id, &wg)
	}

	wg.Wait()

	os.Exit(0)

	return nil
}

func (p *Provider) createRootBucket(name string) (*bolt.Bucket, error) {
	tx, err := p.db.Begin(true)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	bucket, err := tx.CreateBucketIfNotExists([]byte("rack"))
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return bucket, err
}

func (p *Provider) routerCheck() error {
	v, err := p.router.Version()
	if err != nil {
		return err
	}

	if v != "dev" && strings.Compare(v, p.Version) < 0 {
		if err := p.router.Terminate(); err != nil {
			return err
		}

		time.Sleep(3 * time.Second)
	}

	return nil
}

func (p *Provider) routerRegister() error {
	port, err := exec.Command("docker", "inspect", "-f", `{{(index (index .NetworkSettings.Ports "5443/tcp") 0).HostPort}}`, p.Name).CombinedOutput()
	if err != nil {
		return err
	}

	return p.router.RackCreate(p.Name, fmt.Sprintf("tls://127.0.0.1:%s", strings.TrimSpace(string(port))))
}

func (p *Provider) serviceVolumes(app string, volumes []string) ([]string, error) {
	vv := []string{}

	for _, v := range volumes {
		parts := strings.SplitN(v, ":", 2)

		switch len(parts) {
		case 1:
			vv = append(vv, fmt.Sprintf("%s/%s/volumes/%s:%s", p.Volume, app, parts[0], parts[0]))
		case 2:
			vv = append(vv, fmt.Sprintf("%s/%s/volumes/%s:%s", p.Volume, app, parts[0], parts[1]))
		}
	}

	return vv, nil
}
