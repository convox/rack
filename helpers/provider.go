package helpers

import (
	"fmt"

	"github.com/convox/rack/manifest"
	"github.com/convox/rack/options"
	"github.com/convox/rack/structs"
	"github.com/pkg/errors"
)

func AppEnvironment(p structs.Provider, app string) (structs.Environment, error) {
	rs, err := ReleaseLatest(p, app)
	if err != nil {
		return nil, err
	}
	if rs == nil {
		return structs.Environment{}, nil
	}

	env := structs.Environment{}

	if err := env.Load([]byte(rs.Env)); err != nil {
		return nil, err
	}

	return env, nil
}

func AppManifest(p structs.Provider, app string) (*manifest.Manifest, *structs.Release, error) {
	a, err := p.AppGet(app)
	if err != nil {
		return nil, nil, err
	}

	if a.Release == "" {
		return nil, nil, errors.WithStack(fmt.Errorf("no release for app: %s", app))
	}

	return ReleaseManifest(p, app, a.Release)
}

func ReleaseLatest(p structs.Provider, app string) (*structs.Release, error) {
	rs, err := p.ReleaseList(app, structs.ReleaseListOptions{Count: options.Int(1)})
	if err != nil {
		return nil, err
	}

	if len(rs) < 1 {
		return nil, nil
	}

	return p.ReleaseGet(app, rs[0].Id)
}

func ReleaseManifest(p structs.Provider, app, release string) (*manifest.Manifest, *structs.Release, error) {
	r, err := p.ReleaseGet(app, release)
	if err != nil {
		return nil, nil, err
	}

	if r.Build == "" {
		return nil, nil, errors.WithStack(fmt.Errorf("no builds for app: %s", app))
	}

	b, err := p.BuildGet(app, r.Build)
	if err != nil {
		return nil, nil, err
	}

	env := structs.Environment{}

	if err := env.Load([]byte(r.Env)); err != nil {
		return nil, nil, err
	}

	m, err := manifest.Load([]byte(b.Manifest), manifest.Environment(env))
	if err != nil {
		return nil, nil, err
	}

	return m, r, nil
}
