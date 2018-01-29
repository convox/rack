package local

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/convox/rack/helpers"
	"github.com/convox/rack/structs"
	"github.com/pkg/errors"
)

const (
	AppCacheDuration = 5 * time.Minute
)

func (p *Provider) AppCancel(name string) error {
	return fmt.Errorf("unimplemented")
}

func (p *Provider) AppCreate(name string, opts structs.AppCreateOptions) (*structs.App, error) {
	log := p.logger("AppCreate").Append("name=%q", name)

	if p.storageExists(fmt.Sprintf("apps/%s/app.json", name)) {
		return nil, log.Error(fmt.Errorf("app already exists: %s", name))
	}

	app := &structs.App{
		Name:   name,
		Status: "running",
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
	log := p.logger("AppGet").Append("name=%q", name)

	var app structs.App

	if err := p.storageLoad(fmt.Sprintf("apps/%s/app.json", name), &app, AppCacheDuration); err != nil {
		if strings.HasPrefix(err.Error(), "no such key:") {
			return nil, fmt.Errorf("no such app: %s", name)
		}
		return nil, errors.WithStack(log.Error(err))
	}

	return &app, log.Success()
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

	pss, err := p.ProcessList(app, structs.ProcessListOptions{})
	if err != nil {
		return nil, errors.WithStack(log.Error(err))
	}

	r, w := io.Pipe()

	var wg sync.WaitGroup

	for _, ps := range pss {
		wg.Add(1)
		go func(ps structs.Process, wg *sync.WaitGroup) {
			defer wg.Done()
			if pr, err := p.ProcessLogs(app, ps.Id, opts); err == nil {
				helpers.Stream(w, pr)
			}
		}(ps, &wg)
	}

	go func() {
		wg.Wait()
		w.Close()
	}()

	return r, log.Success()
}

func (p *Provider) AppRegistry(app string) (*structs.Registry, error) {
	log := p.logger("AppRegistry").Append("app=%q", app)

	if _, err := p.AppGet(app); err != nil {
		return nil, log.Error(err)
	}

	registry := &structs.Registry{
		Server:   p.Name,
		Username: "",
		Password: "",
	}

	return registry, log.Success()
}

func (p *Provider) AppUpdate(app string, opts structs.AppUpdateOptions) error {
	return fmt.Errorf("unimplemented")
}
