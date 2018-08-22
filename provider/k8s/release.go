package k8s

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/convox/rack/helpers"
	"github.com/convox/rack/options"
	ca "github.com/convox/rack/provider/k8s/pkg/apis/convox/v1"
	"github.com/convox/rack/structs"
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

func (p *Provider) ReleasePromote(app, id string) error {
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

	params := map[string]interface{}{
		"App":       a,
		"Namespace": p.appNamespace(a.Name),
		"Rack":      p.Rack,
		"Release":   r,
		"Services":  m.Services,
	}

	data, err := p.yamlTemplate("router", params)
	if err != nil {
		return err
	}

	items = append(items, data)

	for _, s := range m.Services {
		ps, err := p.ProcessList(app, structs.ProcessListOptions{Release: options.String(a.Release), Service: options.String(s.Name)})
		if err != nil {
			return err
		}

		replicas := helpers.CoalesceInt(len(ps), s.Scale.Count.Min)

		params := map[string]interface{}{
			"App":       a,
			"Env":       e,
			"Manifest":  m,
			"Namespace": p.appNamespace(a.Name),
			"Rack":      p.Rack,
			"Release":   r,
			"Replicas":  replicas,
			"Service":   s,
		}

		data, err := p.yamlTemplate("service", params)
		if err != nil {
			return err
		}

		items = append(items, data)
	}

	fmt.Println("updating")

	tdata := bytes.Join(items, []byte("---\n"))

	// fmt.Printf("string(tdata) = %+v\n", string(tdata))

	cmd := exec.Command("kubectl", "apply", "--prune", fmt.Sprintf("-l system=convox,rack=%s,app=%s", p.Rack, app), "-f", "-")

	cmd.Stdin = bytes.NewReader(tdata)

	data, err = cmd.CombinedOutput()
	if err != nil {
		return errors.New(strings.TrimSpace(string(data)))
	}

	fmt.Printf("string(data) = %+v\n", string(data))

	ns, err := p.Cluster.CoreV1().Namespaces().Get(p.appNamespace(app), am.GetOptions{})
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

	fmt.Println("done")

	return nil
}

func (p *Provider) releaseCreate(r *structs.Release) (*structs.Release, error) {
	c, err := p.convoxClient()
	if err != nil {
		return nil, err
	}

	kr, err := c.Releases(p.appNamespace(r.App)).Create(p.releaseMarshal(r))
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

	kr, err := c.Releases(p.appNamespace(app)).Get(strings.ToLower(id), am.GetOptions{})
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

	krs, err := c.Releases(p.appNamespace(app)).List(am.ListOptions{})
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
			Namespace: p.appNamespace(r.App),
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
			Env:      r.Env,
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

	return r, nil
}
