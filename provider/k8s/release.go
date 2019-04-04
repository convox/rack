package k8s

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/manifest"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	ca "github.com/convox/rack/provider/k8s/pkg/apis/convox/v1"
	ac "k8s.io/api/core/v1"
	am "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (p *Provider) ReleaseCreate(app string, opts structs.ReleaseCreateOptions) (*structs.Release, error) {
	r, err := p.releaseFork(app)
	if err != nil {
		return nil, err
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

	ro, err := p.releaseCreate(r)
	if err != nil {
		return nil, err
	}

	return ro, nil
}

func (p *Provider) ReleaseGet(app, id string) (*structs.Release, error) {
	r, err := p.releaseGet(app, id)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (p *Provider) ReleaseList(app string, opts structs.ReleaseListOptions) (structs.Releases, error) {
	if _, err := p.AppGet(app); err != nil {
		return nil, err
	}

	rs, err := p.releaseList(app)
	if err != nil {
		return nil, err
	}

	sort.Slice(rs, func(i, j int) bool { return rs[j].Created.Before(rs[i].Created) })

	if limit := helpers.DefaultInt(opts.Limit, 10); len(rs) > limit {
		rs = rs[0:limit]
	}

	return rs, nil
}

func (p *Provider) ReleasePromote(app, id string, opts structs.ReleasePromoteOptions) error {
	a, err := p.AppGet(app)
	if err != nil {
		return err
	}

	m, r, err := helpers.ReleaseManifest(p, app, id)
	if err != nil {
		return err
	}

	e := structs.Environment{}
	e.Load([]byte(r.Env))

	items := [][]byte{}

	idles, err := p.Engine.AppIdles(a.Name)
	if err != nil {
		return err
	}

	sps := manifest.Services{}

	for _, s := range m.Services {
		if s.Port.Port > 0 {
			sps = append(sps, s)
		}
	}

	vsh := map[string]bool{}

	for _, s := range m.Services {
		for _, v := range p.volumeSources(app, s.Name, s.Volumes) {
			if !systemVolume(v) {
				vsh[v] = true
			}
		}
	}

	vs := []string{}

	for s := range vsh {
		vs = append(vs, s)
	}

	params := map[string]interface{}{
		"App":       a,
		"Idles":     helpers.DefaultBool(opts.Idle, idles),
		"Manifest":  m,
		"Namespace": p.AppNamespace(a.Name),
		"Rack":      p.Rack,
		"Release":   r,
		"Services":  sps,
		"Volumes":   vs,
	}

	if ca, err := p.Cluster.CoreV1().Secrets("convox-system").Get("ca", am.GetOptions{}); err == nil {
		params["CA"] = base64.StdEncoding.EncodeToString(ca.Data["tls.crt"])

		data, err := p.RenderTemplate("config", params)
		if err != nil {
			return err
		}

		items = append(items, data)
	}

	data, err := p.RenderTemplate("ingress", params)
	if err != nil {
		return err
	}

	items = append(items, data)

	data, err = p.RenderTemplate("volumes", params)
	if err != nil {
		return err
	}

	items = append(items, data)

	for _, r := range m.Resources {
		data, err := p.Engine.ResourceRender(app, r)
		if err != nil {
			return err
		}

		items = append(items, data)
	}

	ss, err := p.ServiceList(app)
	if err != nil {
		return err
	}

	sc := map[string]int{}

	for _, s := range ss {
		sc[s.Name] = s.Count
	}

	var b *structs.Build

	if r.Build != "" {
		bb, err := p.BuildGet(app, r.Build)
		if err != nil {
			return err
		}
		b = bb
	}

	senv, err := p.systemEnvironment(a.Name, r.Id)
	if err != nil {
		return err
	}

	for _, s := range m.Services {
		min := 50
		max := 200

		if s.Agent.Enabled || s.Singleton {
			min = 0
			max = 100
		}

		if opts.Min != nil {
			min = *opts.Min
		}

		if opts.Max != nil {
			max = *opts.Max
		}

		replicas := helpers.CoalesceInt(sc[s.Name], s.Scale.Count.Min)

		params := map[string]interface{}{
			"App":            a,
			"Build":          b,
			"Development":    helpers.DefaultBool(opts.Development, false),
			"Manifest":       m,
			"MaxSurge":       max,
			"MaxUnavailable": 100 - min,
			"Namespace":      p.AppNamespace(a.Name),
			"Password":       p.Password,
			"Rack":           p.Rack,
			"Release":        r,
			"Replicas":       replicas,
			"Rollback":       a.Release,
			"Service":        s,
			"SystemEnv":      senv,
			"Timeout":        helpers.DefaultInt(opts.Timeout, 1800),
		}

		if ip, err := p.Engine.Resolver(); err == nil {
			params["Resolver"] = ip
		}

		data, err := p.RenderTemplate("service", params)
		if err != nil {
			return err
		}

		items = append(items, data)
	}

	tdata := bytes.Join(items, []byte("---\n"))

	// fmt.Printf("string(tdata) = %+v\n", string(tdata))

	out, err := p.Apply(tdata, fmt.Sprintf("system=convox,provider=k8s,scope=release,rack=%s,app=%s", p.Rack, app))
	if err != nil {
		return errors.New(strings.TrimSpace(string(out)))
	}

	ns, err := p.Cluster.CoreV1().Namespaces().Get(p.AppNamespace(app), am.GetOptions{})
	if err != nil {
		return err
	}

	if ns.ObjectMeta.Annotations == nil {
		ns.ObjectMeta.Annotations = map[string]string{}
	}

	ns.Annotations["convox.release"] = r.Id

	if _, err := p.Cluster.CoreV1().Namespaces().Update(ns); err != nil {
		return err
	}

	return nil
}

func (p *Provider) releaseCreate(r *structs.Release) (*structs.Release, error) {
	c, err := p.convoxClient()
	if err != nil {
		return nil, err
	}

	e := structs.Environment{}

	if err := e.Load([]byte(r.Env)); err != nil {
		return nil, err
	}

	s := &ac.Secret{
		ObjectMeta: am.ObjectMeta{
			Name: fmt.Sprintf("release-%s", strings.ToLower(r.Id)),
			Labels: map[string]string{
				"system": "convox",
				"rack":   p.Rack,
				"app":    r.App,
				"type":   "release",
			},
		},
		StringData: e,
	}

	if _, err := p.Cluster.CoreV1().Secrets(p.AppNamespace(r.App)).Create(s); err != nil {
		return nil, err
	}

	kr, err := c.Releases(p.AppNamespace(r.App)).Create(p.releaseMarshal(r))
	if err != nil {
		return nil, err
	}

	return p.releaseUnmarshal(kr)
}

func (p *Provider) releaseGet(app, id string) (*structs.Release, error) {
	c, err := p.convoxClient()
	if err != nil {
		return nil, err
	}

	kr, err := c.Releases(p.AppNamespace(app)).Get(strings.ToLower(id), am.GetOptions{})
	if err != nil {
		return nil, err
	}

	return p.releaseUnmarshal(kr)
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

func (p *Provider) releaseList(app string) (structs.Releases, error) {
	c, err := p.convoxClient()
	if err != nil {
		return nil, err
	}

	krs, err := c.Releases(p.AppNamespace(app)).List(am.ListOptions{})
	if err != nil {
		return nil, err
	}

	rs := structs.Releases{}

	for _, kr := range krs.Items {
		r, err := p.releaseUnmarshal(&kr)
		if err != nil {
			return nil, err
		}

		rs = append(rs, *r)
	}

	return rs, nil
}

func (p *Provider) releaseMarshal(r *structs.Release) *ca.Release {
	return &ca.Release{
		ObjectMeta: am.ObjectMeta{
			Namespace: p.AppNamespace(r.App),
			Name:      strings.ToLower(r.Id),
			Labels: map[string]string{
				"system": "convox",
				"rack":   p.Rack,
				"app":    r.App,
			},
		},
		Spec: ca.ReleaseSpec{
			Build:    r.Build,
			Created:  r.Created.Format(helpers.SortableTime),
			Manifest: r.Manifest,
		},
	}
}

func (p *Provider) releaseUnmarshal(kr *ca.Release) (*structs.Release, error) {
	created, err := time.Parse(helpers.SortableTime, kr.Spec.Created)
	if err != nil {
		return nil, err
	}

	r := &structs.Release{
		App:      kr.ObjectMeta.Labels["app"],
		Build:    kr.Spec.Build,
		Created:  created,
		Env:      kr.Spec.Env,
		Id:       strings.ToUpper(kr.ObjectMeta.Name),
		Manifest: kr.Spec.Manifest,
	}

	if s, err := p.Cluster.CoreV1().Secrets(p.AppNamespace(r.App)).Get(fmt.Sprintf("release-%s", kr.ObjectMeta.Name), am.GetOptions{}); err == nil {
		e := structs.Environment{}

		for k, v := range s.Data {
			e[k] = string(v)
		}

		r.Env = e.String()
	}

	return r, nil
}
