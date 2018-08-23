package local

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/convox/rack/pkg/cache"
	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/structs"
	"github.com/pkg/errors"
)

const (
	ReleaseCacheDuration = 1 * time.Hour
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func (p *Provider) ReleaseCreate(app string, opts structs.ReleaseCreateOptions) (*structs.Release, error) {
	log := p.logger("ReleaseCreate").Append("app=%q", app)

	r, err := p.releaseFork(app)
	if err != nil {
		return nil, errors.WithStack(log.Error(err))
	}

	if opts.Build != nil {
		r.Build = *opts.Build
	}

	if opts.Env != nil {
		r.Env = *opts.Env
	}

	if r.Build != "" {
		b, err := p.BuildGet(app, r.Build)
		if err != nil {
			return nil, err
		}

		r.Manifest = b.Manifest
	}

	if err := p.storageStore(fmt.Sprintf("apps/%s/releases/%s/release.json", app, r.Id), r); err != nil {
		return nil, errors.WithStack(log.Error(err))
	}

	return r, log.Success()
}

func (p *Provider) ReleaseGet(app, id string) (*structs.Release, error) {
	log := p.logger("ReleaseGet").Append("app=%q id=%q", app, id)

	var r *structs.Release

	if err := p.storageLoad(fmt.Sprintf("apps/%s/releases/%s/release.json", app, id), &r, ReleaseCacheDuration); err != nil {
		if strings.Contains(err.Error(), "no such key") {
			return nil, fmt.Errorf("release not found")
		}
		return nil, errors.WithStack(log.Error(err))
	}
	if r == nil {
		return nil, log.Error(fmt.Errorf("could not find release: %s", id))
	}

	return r, log.Success()
}

func (p *Provider) ReleaseList(app string, opts structs.ReleaseListOptions) (structs.Releases, error) {
	log := p.logger("ReleaseList").Append("app=%q", app)

	if _, err := p.AppGet(app); err != nil {
		return nil, log.Error(err)
	}

	ids, err := p.storageList(fmt.Sprintf("apps/%s/releases", app))
	if err != nil {
		return nil, errors.WithStack(log.Error(err))
	}

	releases := make(structs.Releases, len(ids))

	for i, id := range ids {
		release, err := p.ReleaseGet(app, id)
		if err != nil {
			return nil, log.Error(err)
		}

		releases[i] = *release
	}

	sort.Slice(releases, func(i, j int) bool { return releases[j].Created.Before(releases[i].Created) })

	limit := 10

	if opts.Limit != nil {
		limit = *opts.Limit
	}

	if len(releases) > limit {
		releases = releases[0:limit]
	}

	return releases, log.Success()
}

func (p *Provider) ReleasePromote(app, id string) error {
	log := p.logger("ReleasePromote").Append("app=%q id=%q", app, id)

	a, err := p.AppGet(app)
	if err != nil {
		return log.Error(err)
	}

	// clear current release cache so its no longer "active"
	cache.Clear("storage", fmt.Sprintf("apps/%s/releases/%s/release.json", app, a.Release))

	r, err := p.ReleaseGet(app, id)

	if r.Build == "" {
		return fmt.Errorf("no build for release: %s", id)
	}

	if err := p.storageStore(fmt.Sprintf("apps/%s/releases/%s/release.json", app, id), r); err != nil {
		return errors.WithStack(log.Error(err))
	}

	a.Release = r.Id
	a.Sleep = false

	if err := p.storageStore(fmt.Sprintf("apps/%s/app.json", a.Name), a); err != nil {
		return errors.WithStack(log.Error(err))
	}

	if err := p.converge(app); err != nil {
		return errors.WithStack(log.Error(err))
	}

	if err := p.storageStore(fmt.Sprintf("apps/%s/releases/%s/release.json", app, id), r); err != nil {
		return errors.WithStack(log.Error(err))
	}

	return log.Success()
}

func (p *Provider) releaseFork(app string) (*structs.Release, error) {
	r := &structs.Release{
		Id:      helpers.Id("R", 10),
		App:     app,
		Created: time.Now().UTC(),
	}

	rs, err := p.ReleaseList(app, structs.ReleaseListOptions{Limit: options.Int(1)})
	if err != nil {
		return nil, err
	}

	if len(rs) > 0 {
		r.Build = rs[0].Build
		r.Env = rs[0].Env
	}

	return r, nil
}
