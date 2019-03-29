package k8s

import (
	"fmt"
	"strings"

	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/structs"
	am "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TODO finish
func (p *Provider) ServiceList(app string) (structs.Services, error) {
	lopts := am.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s,type=service", app),
	}

	ds, err := p.Cluster.AppsV1().Deployments(p.AppNamespace(app)).List(lopts)
	if err != nil {
		return nil, err
	}

	a, err := p.AppGet(app)
	if err != nil {
		return nil, err
	}

	if a.Release == "" {
		return structs.Services{}, nil
	}

	m, _, err := helpers.ReleaseManifest(p, app, a.Release)
	if err != nil {
		return nil, err
	}

	ss := structs.Services{}

	for _, d := range ds.Items {
		cs := d.Spec.Template.Spec.Containers

		if len(cs) != 1 || cs[0].Name != "main" {
			return nil, fmt.Errorf("unexpected containers for service: %s", d.ObjectMeta.Name)
		}

		// fmt.Printf("d.Spec = %+v\n", d.Spec)
		// fmt.Printf("d.Status = %+v\n", d.Status)

		s := structs.Service{
			Count: int(helpers.DefaultInt32(d.Spec.Replicas, 0)),
			Name:  d.ObjectMeta.Name,
			Ports: []structs.ServicePort{},
		}

		if len(cs[0].Ports) == 1 {
			// i, err := p.Cluster.ExtensionsV1beta1().Ingresses(p.AppNamespace(app)).Get(app, am.GetOptions{})
			// if err != nil {
			//   return nil, err
			// }

			ms, err := m.Service(d.ObjectMeta.Name)
			if err != nil {
				return nil, err
			}

			s.Domain = p.Engine.ServiceHost(app, *ms)

			// s.Domain = fmt.Sprintf("%s.%s", p.Engine.ServiceHost(app, s.Name), helpers.CoalesceString(i.Annotations["convox.domain"], i.Labels["rack"]))

			// if domain, ok := i.Annotations["convox.domain"]; ok {
			//   s.Domain += fmt.Sprintf(".%s", domain)
			// }

			s.Ports = append(s.Ports, structs.ServicePort{Balancer: 443, Container: int(cs[0].Ports[0].ContainerPort)})
		}

		ss = append(ss, s)
	}

	return ss, nil
}

func (p *Provider) ServiceUpdate(app, name string, opts structs.ServiceUpdateOptions) error {
	d, err := p.Cluster.AppsV1().Deployments(p.AppNamespace(app)).Get(name, am.GetOptions{})
	if err != nil {
		return err
	}

	if opts.Count != nil {
		c := int32(*opts.Count)
		d.Spec.Replicas = &c
	}

	if _, err := p.Cluster.AppsV1().Deployments(p.AppNamespace(app)).Update(d); err != nil {
		return err
	}

	return nil
}

func (p *Provider) serviceInstall(app, release, service string) error {
	a, err := p.AppGet(app)
	if err != nil {
		return err
	}

	m, r, err := helpers.ReleaseManifest(p, app, release)
	if err != nil {
		return err
	}

	s, err := m.Service(service)
	if err != nil {
		return err
	}

	if s.Port.Port == 0 {
		return nil
	}

	params := map[string]interface{}{
		"Namespace": p.AppNamespace(a.Name),
		"Release":   r,
		"Service":   s,
	}

	if out, err := p.ApplyTemplate("port", fmt.Sprintf("system=convox,provider=k8s,scope=port,rack=%s,app=%s,service=%s", p.Rack, app, service), params); err != nil {
		return fmt.Errorf("update error: %s: %s", err, strings.TrimSpace(string(out)))
	}

	return nil
}
