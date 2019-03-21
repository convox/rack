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
	if _, err := p.Cluster.CoreV1().Namespaces().Get(p.AppNamespace(name), am.GetOptions{}); !ae.IsNotFound(err) {
		return nil, fmt.Errorf("app already exists: %s", name)
	}

	params := map[string]interface{}{
		"Name":      name,
		"Namespace": p.AppNamespace(name),
		"Rack":      p.Rack,
	}

	if out, err := p.ApplyTemplate("app", fmt.Sprintf("system=convox,provider=k8s,scope=app,rack=%s,app=%s", p.Rack, name), params); err != nil {
		return nil, fmt.Errorf("create error: %s", string(out))
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

	// if err := p.Storage.Clear(fmt.Sprintf("apps/%s", name)); err != nil {
	//   return err
	// }

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

	// fmt.Printf("ns = %+v\n", ns)

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

func (p *Provider) AppStatus(app string) (string, error) {
	// ps, err := p.Cluster.CoreV1().Pods(p.AppNamespace(app)).List(am.ListOptions{})
	// if err != nil {
	//   return "", err
	// }

	// for _, p := range ps.Items {
	//   for _, c := range p.Status.ContainerStatuses {
	//     if c.State.Waiting != nil && c.State.Waiting.Reason == "CrashLoopBackOff" {
	//       return "crashing", nil
	//     }
	//   }
	// }

	ds, err := p.Cluster.AppsV1().Deployments(p.AppNamespace(app)).List(am.ListOptions{})
	if err != nil {
		return "", err
	}

	for _, d := range ds.Items {
		switch {
		case d.Spec.Replicas != nil && d.Status.UpdatedReplicas < *d.Spec.Replicas:
			return "updating", nil
		case d.Status.Replicas > d.Status.UpdatedReplicas:
			return "updating", nil
		case d.Status.AvailableReplicas < d.Status.UpdatedReplicas:
			return "updating", nil
		}
	}

	return p.Engine.AppStatus(app)
}

func (p *Provider) AppUpdate(name string, opts structs.AppUpdateOptions) error {
	return fmt.Errorf("unimplemented")
}

func (p *Provider) appFromNamespace(ns ac.Namespace) (*structs.App, error) {
	status := "unknown"

	name := helpers.CoalesceString(ns.Labels["app"], ns.Labels["name"])

	switch ns.Status.Phase {
	case "Terminating":
		status = "deleting"
	default:
		s, err := p.AppStatus(name)
		if err != nil {
			return nil, err
		}
		status = s
	}

	a := &structs.App{
		Generation: "2",
		Name:       name,
		Release:    ns.Annotations["convox.release"],
		Status:     status,
	}

	return a, nil
}
