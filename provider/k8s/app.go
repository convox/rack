package k8s

import (
	"fmt"
	"io"

	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/structs"
	ac "k8s.io/api/core/v1"
	ae "k8s.io/apimachinery/pkg/api/errors"
	am "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (p *Provider) AppCancel(name string) error {
	return fmt.Errorf("unimplemented")
}

func (p *Provider) AppCreate(name string, opts structs.AppCreateOptions) (*structs.App, error) {
	if err := p.appNameValidate(name); err != nil {
		return nil, err
	}

	params := map[string]interface{}{
		"Name":      name,
		"Namespace": p.AppNamespace(name),
		"Rack":      p.Rack,
	}

	data, err := p.RenderTemplate("app/app", params)
	if err != nil {
		return nil, err
	}

	if err := p.ApplyWait(p.AppNamespace(name), "app", "", data, fmt.Sprintf("system=convox,provider=k8s,rack=%s,app=%s", p.Rack, name), 30); err != nil {
		return nil, err
	}

	return p.AppGet(name)
}

func (p *Provider) AppDelete(name string) error {
	if _, err := p.AppGet(name); err != nil {
		return err
	}

	if err := p.Cluster.CoreV1().Namespaces().Delete(p.AppNamespace(name), nil); err != nil {
		return err
	}

	return nil
}

func (p *Provider) AppGet(name string) (*structs.App, error) {
	ns, err := p.Cluster.CoreV1().Namespaces().Get(p.AppNamespace(name), am.GetOptions{})
	if ae.IsNotFound(err) {
		return nil, fmt.Errorf("app not found: %s", name)
	}
	if err != nil {
		return nil, err
	}

	a, err := p.appFromNamespace(*ns)
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (p *Provider) AppList() (structs.Apps, error) {
	ns, err := p.Cluster.CoreV1().Namespaces().List(am.ListOptions{
		LabelSelector: fmt.Sprintf("system=convox,rack=%s,type=app", p.Rack),
	})
	if err != nil {
		return nil, err
	}

	as := structs.Apps{}

	for _, n := range ns.Items {
		a, err := p.appFromNamespace(n)
		if err != nil {
			return nil, err
		}

		as = append(as, *a)
	}

	return as, nil
}

func (p *Provider) AppLogs(name string, opts structs.LogsOptions) (io.ReadCloser, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) AppMetrics(name string, opts structs.MetricsOptions) (structs.Metrics, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) AppNamespace(app string) string {
	switch app {
	case "system":
		return "convox-system"
	case "rack":
		return p.Rack
	default:
		return fmt.Sprintf("%s-%s", p.Rack, app)
	}
}

func (p *Provider) AppUpdate(name string, opts structs.AppUpdateOptions) error {
	return fmt.Errorf("unimplemented")
}

func (p *Provider) appFromNamespace(ns ac.Namespace) (*structs.App, error) {
	name := helpers.CoalesceString(ns.Labels["app"], ns.Labels["name"])

	as, release, err := p.atom.Status(ns.Name, "app")
	if err != nil {
		return nil, err
	}

	status, err := p.Engine.AppStatus(name)
	if err != nil {
		return nil, err
	}

	switch status {
	case "running", "unknown":
		status = helpers.AtomStatus(as)
	}

	a := &structs.App{
		Generation: "2",
		Name:       name,
		Release:    release,
		Status:     status,
	}

	switch ns.Status.Phase {
	case "Terminating":
		a.Status = "deleting"
	}

	return a, nil
}

func (p *Provider) appNameValidate(name string) error {
	switch name {
	case "rack", "system":
		return fmt.Errorf("app name is reserved")
	}

	if _, err := p.Cluster.CoreV1().Namespaces().Get(p.AppNamespace(name), am.GetOptions{}); !ae.IsNotFound(err) {
		return fmt.Errorf("app already exists: %s", name)
	}

	return nil
}
