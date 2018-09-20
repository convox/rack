package local

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/pkg/errors"
)

const (
	AppCacheDuration = 5 * time.Minute
)

func (p *Provider) AppCancel(name string) error {
	return fmt.Errorf("cannot cancel deploys on a local rack")
}

func (p *Provider) AppCreate(name string, opts structs.AppCreateOptions) (*structs.App, error) {
	log := p.logger("AppCreate").Append("name=%q", name)

	if p.storageExists(fmt.Sprintf("apps/%s/app.json", name)) {
		return nil, log.Error(fmt.Errorf("app already exists: %s", name))
	}

	app := &structs.App{
		Name:       name,
		Generation: "2",
		Status:     "running",
	}

	if err := p.storageStore(fmt.Sprintf("apps/%s/app.json", app.Name), app); err != nil {
		return nil, errors.WithStack(log.Error(err))
	}

	return app, log.Success()
}

func (p *Provider) AppDelete(app string) error {
	log := p.logger("AppDelete").Append("app=%q", app)

	if _, err := p.AppGet(app); err != nil {
		return log.Error(err)
	}

	pss, err := p.ProcessList(app, structs.ProcessListOptions{})
	if err != nil {
		return errors.WithStack(log.Error(err))
	}

	for _, ps := range pss {
		if err := p.ProcessStop(app, ps.Id); err != nil {
			return errors.WithStack(log.Error(err))
		}
	}

	if err := p.storageDeleteAll(fmt.Sprintf("apps/%s", app)); err != nil {
		return errors.WithStack(log.Error(err))
	}

	return log.Success()
}

func (p *Provider) AppGet(name string) (*structs.App, error) {
	var app structs.App

	if err := p.storageLoad(fmt.Sprintf("apps/%s/app.json", name), &app, AppCacheDuration); err != nil {
		if strings.HasPrefix(err.Error(), "no such key:") {
			return nil, fmt.Errorf("no such app: %s", name)
		}
		return nil, errors.WithStack(err)
	}

	return &app, nil
}

func (p *Provider) AppList() (structs.Apps, error) {
	log := p.logger("AppList")

	names, err := p.storageList("apps")
	if err != nil {
		return nil, errors.WithStack(log.Error(err))
	}

	apps := make(structs.Apps, len(names))

	for i, name := range names {
		app, err := p.AppGet(name)
		if err != nil {
			return nil, errors.WithStack(log.Error(err))
		}

		apps[i] = *app
	}

	sort.Slice(apps, func(i, j int) bool { return apps[i].Name < apps[j].Name })

	return apps, log.Successf("count=%d", len(apps))
}

func (p *Provider) AppLogs(app string, opts structs.LogsOptions) (io.ReadCloser, error) {
	log := p.logger("AppLogs").Append("app=%q", app)

	if _, err := p.AppGet(app); err != nil {
		return nil, log.Error(err)
	}

	r, w := io.Pipe()

	go func() {
		pids := map[string]bool{}

		var wg sync.WaitGroup
		done := false

		go func() {
			time.Sleep(5 * time.Second)
			wg.Wait()
			done = true
			w.Close()
		}()

		for {
			if done {
				break
			}

			pss, err := p.ProcessList(app, structs.ProcessListOptions{})
			if err != nil {
				log.Error(err)
				continue
			}

			for _, ps := range pss {
				popts := opts
				popts.Since = options.Duration(60 * time.Minute)
				if _, ok := pids[ps.Id]; !ok {
					go p.streamProcessLogs(app, ps.Id, popts, w, &wg)
					pids[ps.Id] = true
				}
			}

			time.Sleep(1 * time.Second)
		}
	}()

	return r, log.Success()
}

func (p *Provider) streamProcessLogs(app, pid string, opts structs.LogsOptions, w io.Writer, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()
	r, err := p.ProcessLogs(app, pid, opts)
	if err != nil {
		return
	}
	helpers.Stream(w, r)
}

func (p *Provider) AppRegistry(app string) (*structs.Registry, error) {
	log := p.logger("AppRegistry").Append("app=%q", app)

	if _, err := p.AppGet(app); err != nil {
		return nil, log.Error(err)
	}

	registry := &structs.Registry{
		Server:   p.Rack,
		Username: "",
		Password: "",
	}

	return registry, log.Success()
}

func (p *Provider) AppUpdate(app string, opts structs.AppUpdateOptions) error {
	log := p.logger("AppUpdate").Append("app=%q", app)

	if opts.Sleep != nil {
		a, err := p.AppGet(app)
		if err != nil {
			return errors.WithStack(log.Error(err))
		}

		a.Sleep = *opts.Sleep

		if err := p.storageStore(fmt.Sprintf("apps/%s/app.json", app), a); err != nil {
			return errors.WithStack(log.Error(err))
		}

		if err := p.converge(app); err != nil {
			return errors.WithStack(log.Error(err))
		}
	}

	return log.Success()
}
