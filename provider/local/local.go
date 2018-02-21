package local

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/boltdb/bolt"
	"github.com/convox/logger"
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

	ctx  context.Context
	db   *bolt.DB
	logs *logger.Logger
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

	db, err := bolt.Open(filepath.Join(p.Root, "convox.db"), 0600, nil)
	if err != nil {
		return err
	}

	p.db = db

	if _, err := p.createRootBucket("rack"); err != nil {
		return err
	}

	if err := p.checkRouter(); err != nil {
		return err
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

func (p *Provider) checkRouter() error {
	if p.Router == "none" {
		return nil
	}

	c := http.Client{
		Timeout: 2 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	res, err := c.Get(fmt.Sprintf("https://%s/version", p.Router))
	if err != nil {
		return fmt.Errorf("unable to register with router")
	}

	defer res.Body.Close()

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	var v struct {
		Version string
	}

	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	if v.Version != "dev" && strings.Compare(v.Version, p.Version) < 0 {
		c.PostForm(fmt.Sprintf("https://%s/terminate", p.Router), nil)
	}

	return nil
}
