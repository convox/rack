package k8s

import (
	"fmt"

	"github.com/convox/rack/helpers"
	"github.com/convox/rack/structs"
	am "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TODO finish
func (p *Provider) ServiceList(app string) (structs.Services, error) {
	ds, err := p.Cluster.AppsV1().Deployments(p.appNamespace(app)).List(am.ListOptions{})
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
			Count:  int(helpers.DefaultInt32(d.Spec.Replicas, 0)),
			Domain: p.HostFunc(app, d.ObjectMeta.Name),
			Name:   d.ObjectMeta.Name,
			Ports:  []structs.ServicePort{},
		}

		if len(cs[0].Ports) == 1 {
			i, err := p.Cluster.ExtensionsV1beta1().Ingresses(p.appNamespace(app)).Get(app, am.GetOptions{})
			if err != nil {
				return nil, err
			}

			s.Domain = fmt.Sprintf("%s.%s", p.HostFunc(app, s.Name), helpers.CoalesceString(i.Annotations["convox.domain"], i.Labels["rack"]))

			// if domain, ok := i.Annotations["convox.domain"]; ok {
			//   s.Domain += fmt.Sprintf(".%s", domain)
			// }

			s.Ports = append(s.Ports, structs.ServicePort{Balancer: 80, Container: int(cs[0].Ports[0].ContainerPort)})
			s.Ports = append(s.Ports, structs.ServicePort{Balancer: 443, Container: int(cs[0].Ports[0].ContainerPort)})
		}

		ss = append(ss, s)
	}

	return ss, nil
}

func (p *Provider) ServiceUpdate(app, name string, opts structs.ServiceUpdateOptions) error {
	d, err := p.Cluster.AppsV1().Deployments(p.appNamespace(app)).Get(name, am.GetOptions{})
	if err != nil {
		return err
	}

	if opts.Count != nil {
		c := int32(*opts.Count)
		d.Spec.Replicas = &c
	}

	if _, err := p.Cluster.AppsV1().Deployments(p.appNamespace(app)).Update(d); err != nil {
		return err
	}

	return nil
}
