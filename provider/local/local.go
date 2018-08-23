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
	"github.com/convox/rack/pkg/router"
	"github.com/convox/rack/structs"
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

type Provider struct {
	Combined  bool
	Container string
	Image     string
	Rack      string
	Root      string
	Router    string
	Test      bool
	Version   string
	Volume    string

	ctx    context.Context
	db     *bolt.DB
	logs   *logger.Logger
	router *router.Client
}

func FromEnv() (*Provider, error) {
	p := &Provider{
		Combined: os.Getenv("COMBINED") == "true",
		Image:    coalesce(os.Getenv("IMAGE"), "convox/rack"),
		Rack:     coalesce(os.Getenv("RACK"), "convox"),
		Root:     "/var/convox",
		Router:   coalesce(os.Getenv("ROUTER"), "10.42.0.0"),
		Test:     os.Getenv("TEST") == "true",
		Version:  coalesce(os.Getenv("VERSION"), "latest"),
		Volume:   coalesce(os.Getenv("VOLUME"), "/var/convox"),
		logs:     logger.NewWriter("", ioutil.Discard),
	}

	host, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	p.Container = host

	image, err := p.rackImage()
	if err != nil {
		return p, nil
	}

	p.Image = coalesce(os.Getenv("IMAGE"), image)

	return p, nil
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

	db, err := bolt.Open(filepath.Join(p.Root, fmt.Sprintf("%s.db", p.Rack)), 0600, nil)
	if err != nil {
		return err
	}

	p.db = db

	if _, err := p.createRootBucket("rack"); err != nil {
		return err
	}

	if p.Router != "" {
		p.router = router.NewClient(coalesce(p.Router, "10.42.0.0"))

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
		"convox.rack": p.Rack,
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

func (p *Provider) rackImage() (string, error) {
	image, err := exec.Command("docker", "inspect", "-f", "{{.Config.Image}}", p.Container).CombinedOutput()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(image)), nil
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
	port, err := exec.Command("docker", "inspect", "-f", `{{(index (index .NetworkSettings.Ports "5443/tcp") 0).HostPort}}`, p.Container).CombinedOutput()
	if err != nil {
		return err
	}

	return p.router.RackCreate(p.Rack, fmt.Sprintf("tls://127.0.0.1:%s", strings.TrimSpace(string(port))))
}

func systemVolume(volume string) bool {
	switch volume {
	case "/var/run/docker.sock":
		return true
	}

	return false
}

func (p *Provider) serviceVolumes(app string, volumes []string) ([]string, error) {
	vv := []string{}

	for _, v := range volumes {
		parts := strings.SplitN(v, ":", 2)

		from := parts[0]

		if !systemVolume(from) {
			from = fmt.Sprintf("%s/%s/volumes/%s", p.Volume, app, from)
		}

		switch len(parts) {
		case 1:
			vv = append(vv, fmt.Sprintf("%s:%s", from, parts[0]))
		case 2:
			vv = append(vv, fmt.Sprintf("%s:%s", from, parts[1]))
		}
	}

	return vv, nil
}
