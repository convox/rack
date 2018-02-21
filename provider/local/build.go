package local

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/convox/rack/options"
	"github.com/convox/rack/structs"
	"github.com/pkg/errors"
)

const (
	BuildCacheDuration = 1 * time.Second
)

var buildUpdateLock sync.Mutex

func (p *Provider) BuildCreate(app, method, url string, opts structs.BuildCreateOptions) (*structs.Build, error) {
	log := p.logger("BuildCreate").Append("app=%q url=%q", app, url)

	a, err := p.AppGet(app)
	if err != nil {
		return nil, log.Error(err)
	}

	b := structs.NewBuild(app)

	b.Started = time.Now()

	if err := p.storageStore(fmt.Sprintf("apps/%s/builds/%s", app, b.Id), b); err != nil {
		return nil, errors.WithStack(log.Error(err))
	}

	auth, err := p.buildAuth()
	if err != nil {
		return nil, errors.WithStack(log.Error(err))
	}

	sys, err := p.SystemGet()
	if err != nil {
		return nil, errors.WithStack(log.Error(err))
	}

	buildUpdateLock.Lock()
	defer buildUpdateLock.Unlock()

	cache := true

	if opts.Cache != nil {
		cache = *opts.Cache
	}

	pid, err := p.ProcessStart(app, structs.ProcessRunOptions{
		Command: options.String(fmt.Sprintf("build -method %s -cache %t", method, cache)),
		Environment: map[string]string{
			"BUILD_APP":        app,
			"BUILD_AUTH":       string(auth),
			"BUILD_GENERATION": "2",
			"BUILD_ID":         b.Id,
			"BUILD_URL":        url,
			"PROVIDER":         "local",
		},
		Name:    options.String(fmt.Sprintf("%s-build-%s", app, b.Id)),
		Image:   options.String(sys.Image),
		Release: options.String(a.Release),
		Service: options.String("build"),
		Volumes: map[string]string{
			p.Volume:               "/var/convox",
			"/var/run/docker.sock": "/var/run/docker.sock",
		},
	})
	if err != nil {
		return nil, errors.WithStack(log.Error(err))
	}

	b, err = p.BuildGet(app, b.Id)
	if err != nil {
		return nil, errors.WithStack(log.Error(err))
	}

	b.Process = pid
	b.Status = "running"

	if err := p.storageStore(fmt.Sprintf("apps/%s/builds/%s", app, b.Id), b); err != nil {
		return nil, errors.WithStack(log.Error(err))
	}

	return b, log.Successf("id=%s", b.Id)
}

func (p *Provider) BuildExport(app, id string, w io.Writer) error {
	return fmt.Errorf("unimplemented")
}

func (p *Provider) BuildGet(app, id string) (*structs.Build, error) {
	log := p.logger("BuildGet").Append("app=%q id=%q", app, id)

	var b *structs.Build

	if err := p.storageLoad(fmt.Sprintf("apps/%s/builds/%s", app, id), &b, BuildCacheDuration); err != nil {
		if strings.HasPrefix(err.Error(), "no such key:") {
			return nil, log.Error(fmt.Errorf("no such build: %s", id))
		} else {
			return nil, errors.WithStack(log.Error(err))
		}
	}

	return b, log.Success()
}

func (p *Provider) BuildImport(app string, r io.Reader) (*structs.Build, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) BuildList(app string, opts structs.BuildListOptions) (structs.Builds, error) {
	log := p.logger("BuildList").Append("app=%q", app)

	ids, err := p.storageList(fmt.Sprintf("apps/%s/builds", app))
	if err != nil {
		return nil, errors.WithStack(log.Error(err))
	}

	builds := make(structs.Builds, len(ids))

	for i, id := range ids {
		build, err := p.BuildGet(app, id)
		if err != nil {
			return nil, errors.WithStack(log.Error(err))
		}

		builds[i] = *build
	}

	sort.Slice(builds, func(i, j int) bool { return builds[i].Started.After(builds[j].Started) })

	if opts.Count != nil && len(builds) > *opts.Count {
		builds = builds[0:(*opts.Count)]
	}

	return builds, log.Success()
}

func (p *Provider) BuildLogs(app, id string, opts structs.LogsOptions) (io.ReadCloser, error) {
	log := p.logger("BuildLogs").Append("app=%q id=%q", app, id)

	build, err := p.BuildGet(app, id)
	if err != nil {
		return nil, log.Error(err)
	}

	fmt.Printf("build = %+v\n", build)

	switch build.Status {
	case "created", "running":
		fmt.Println("a")
		log.Success()
		return p.ProcessLogs(app, build.Process, structs.LogsOptions{Follow: true, Prefix: false})
	default:
		fmt.Println("b")
		log.Success()
		return p.ObjectFetch(app, fmt.Sprintf("build/%s/logs", id))
	}
}

func (p *Provider) BuildUpdate(app, id string, opts structs.BuildUpdateOptions) (*structs.Build, error) {
	buildUpdateLock.Lock()
	defer buildUpdateLock.Unlock()

	log := p.logger("BuildUpdate").Append("app=%q id=%q", app, id)

	b, err := p.BuildGet(app, id)
	if err != nil {
		return nil, log.Error(err)
	}

	if opts.Ended != nil {
		b.Ended = *opts.Ended
	}

	if opts.Logs != nil {
		b.Logs = *opts.Logs
	}

	if opts.Manifest != nil {
		b.Manifest = *opts.Manifest
	}

	if opts.Release != nil {
		b.Release = *opts.Release
	}

	if opts.Started != nil {
		b.Started = *opts.Started
	}

	if opts.Status != nil {
		b.Status = *opts.Status
	}

	if err := p.storageStore(fmt.Sprintf("apps/%s/builds/%s", app, id), b); err != nil {
		return nil, errors.WithStack(log.Error(err))
	}

	return b, log.Success()
}

func (p *Provider) buildAuth() (string, error) {
	type authEntry struct {
		Username string
		Password string
	}

	auth := map[string]authEntry{}

	registries, err := p.RegistryList()
	if err != nil {
		return "", err
	}

	for _, r := range registries {
		auth[r.Server] = authEntry{
			Username: r.Username,
			Password: r.Password,
		}
	}

	data, err := json.Marshal(auth)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
