package k8s

import (
	"fmt"
	"io"

	"github.com/convox/rack/pkg/structs"
	ac "k8s.io/api/core/v1"
	ae "k8s.io/apimachinery/pkg/api/errors"
	am "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (p *Provider) AppCancel(name string) error {
	return fmt.Errorf("unimplemented")
}

func (p *Provider) AppCreate(name string, opts structs.AppCreateOptions) (*structs.App, error) {
	_, err := p.Cluster.CoreV1().Namespaces().Create(&ac.Namespace{
		ObjectMeta: am.ObjectMeta{
			Name: p.appNamespace(name),
			Labels: map[string]string{
				"system": "convox",
				"rack":   p.Rack,
				"type":   "app",
				"name":   name,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return p.AppGet(name)
}

func (p *Provider) AppGet(name string) (*structs.App, error) {
	ns, err := p.Cluster.CoreV1().Namespaces().Get(p.appNamespace(name), am.GetOptions{})
	if ae.IsNotFound(err) {
		return nil, fmt.Errorf("app not found: %s", name)
	}
	if err != nil {
		return nil, err
	}

	a := appFromNamespace(*ns)

	return &a, nil
}

func (p *Provider) AppDelete(name string) error {
	if _, err := p.AppGet(name); err != nil {
		return err
	}

	if err := p.Cluster.CoreV1().Namespaces().Delete(p.appNamespace(name), nil); err != nil {
		return err
	}

	// if err := p.Storage.Clear(fmt.Sprintf("apps/%s", name)); err != nil {
	//   return err
	// }

	return nil
}

func (p *Provider) AppList() (structs.Apps, error) {
	ns, err := p.Cluster.CoreV1().Namespaces().List(am.ListOptions{
		LabelSelector: fmt.Sprintf("system=convox,rack=%s,type=app", p.Rack),
	})
	if err != nil {
		return nil, err
	}

	// fmt.Printf("ns = %+v\n", ns)

	as := structs.Apps{}

	for _, n := range ns.Items {
		as = append(as, appFromNamespace(n))
	}

	return as, nil
}

func (p *Provider) AppLogs(name string, opts structs.LogsOptions) (io.ReadCloser, error) {
	r, w := io.Pipe()

	go p.streamAppLogs(w, name, opts)

	return r, nil
}

func (p *Provider) AppUpdate(name string, opts structs.AppUpdateOptions) error {
	return fmt.Errorf("unimplemented")
}

func (p *Provider) appNamespace(app string) string {
	if app == "system" {
		return p.Rack
	}

	return fmt.Sprintf("%s-%s", p.Rack, app)
}

func (p *Provider) streamAppLogs(w io.WriteCloser, app string, opts structs.LogsOptions) {
	defer w.Close()

	a, err := p.AppGet(app)
	if err != nil {
		return
	}

	pl := func() (structs.Processes, error) {
		pss, err := p.ProcessList(app, structs.ProcessListOptions{})
		if err != nil {
			return nil, err
		}

		pss = processFilter(pss, func(ps structs.Process) bool {
			return ps.Release == a.Release
		})

		return pss, nil
	}

	ch := make(chan error)

	go p.streamProcessListLogs(w, pl, opts, ch)

	for {
		err, ok := <-ch
		if err != nil {
			fmt.Printf("err = %+v\n", err)
		}
		if !ok {
			return
		}
	}
}

func appFromNamespace(ns ac.Namespace) structs.App {
	status := "unknown"

	switch ns.Status.Phase {
	case "Active":
		status = "running"
	case "Terminating":
		status = "deleting"
	}

	return structs.App{
		Name:    ns.Labels["name"],
		Release: ns.Annotations["convox.release"],
		Status:  status,
	}
}
