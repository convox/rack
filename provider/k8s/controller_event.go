package k8s

import (
	"fmt"
	"time"

	ac "k8s.io/api/core/v1"
	am "k8s.io/apimachinery/pkg/apis/meta/v1"
	ic "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type EventController struct {
	Controller *Controller
	Provider   *Provider

	start am.Time
}

func NewEventController(p *Provider) (*EventController, error) {
	pc := &EventController{
		Provider: p,
		start:    am.NewTime(time.Now().UTC()),
	}

	c, err := NewController(p.Rack, "convox-k8s-event", pc)
	if err != nil {
		return nil, err
	}

	pc.Controller = c

	return pc, nil
}

func (c *EventController) Client() kubernetes.Interface {
	return c.Provider.Cluster
}

func (c *EventController) ListOptions(opts *am.ListOptions) {
	// opts.LabelSelector = fmt.Sprintf("system=convox,rack=%s", c.Provider.Rack)
}

func (c *EventController) Run() {
	i := ic.NewFilteredEventInformer(c.Provider.Cluster, ac.NamespaceAll, 0, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, c.ListOptions)

	ch := make(chan error)

	go c.Controller.Run(i, ch)

	for err := range ch {
		fmt.Printf("err = %+v\n", err)
	}
}

func (c *EventController) Start() error {
	return nil
}

func (c *EventController) Stop() error {
	return nil
}

func (c *EventController) Add(obj interface{}) error {
	e, err := assertEvent(obj)
	if err != nil {
		return err
	}

	if e.LastTimestamp.Before(&c.start) {
		return nil
	}

	o := e.InvolvedObject

	kind := fmt.Sprintf("%s/%s", e.InvolvedObject.APIVersion, e.InvolvedObject.Kind)

	switch kind {
	case "/Node":
	case "apps/v1/Deployment":
		d, err := c.Provider.Cluster.ExtensionsV1beta1().Deployments(o.Namespace).Get(o.Name, am.GetOptions{ResourceVersion: o.ResourceVersion})
		if err != nil {
			return err
		}

		if app := d.ObjectMeta.Labels["app"]; app != "" {
			if err := c.Provider.systemLog(app, d.Name, e.LastTimestamp.Time, e.Message); err != nil {
				return err
			}
		}
	case "apps/v1/ReplicaSet":
		rs, err := c.Provider.Cluster.AppsV1().ReplicaSets(o.Namespace).Get(o.Name, am.GetOptions{ResourceVersion: o.ResourceVersion})
		if err != nil {
			return err
		}

		if app := rs.ObjectMeta.Labels["app"]; app != "" {
			if err := c.Provider.systemLog(app, rs.Name, e.LastTimestamp.Time, e.Message); err != nil {
				return err
			}
		}
	case "autoscaling/v2beta2/HorizontalPodAutoscaler":
		a, err := c.Provider.Cluster.AutoscalingV1().HorizontalPodAutoscalers(o.Namespace).Get(o.Name, am.GetOptions{ResourceVersion: o.ResourceVersion})
		if err != nil {
			return err
		}

		if app := a.ObjectMeta.Labels["app"]; app != "" {
			if err := c.Provider.systemLog(app, a.Name, e.LastTimestamp.Time, e.Message); err != nil {
				return err
			}
		}
	case "v1/ConfigMap":
	case "v1/Pod":
		switch e.Reason {
		case "Killing":
		default:
			p, err := c.Provider.Cluster.CoreV1().Pods(o.Namespace).Get(o.Name, am.GetOptions{ResourceVersion: o.ResourceVersion})
			if err != nil {
				return err
			}

			if app := p.ObjectMeta.Labels["app"]; app != "" {
				if err := c.Provider.systemLog(app, p.Name, e.LastTimestamp.Time, e.Message); err != nil {
					return err
				}
			}
		}
	default:
		fmt.Printf("  unhandled type: %s: %s\n", kind, e.Message)
	}

	return nil
}

func (c *EventController) Delete(obj interface{}) error {
	return nil
}

func (c *EventController) Update(prev, cur interface{}) error {
	return nil
}

func assertEvent(v interface{}) (*ac.Event, error) {
	e, ok := v.(*ac.Event)
	if !ok {
		return nil, fmt.Errorf("could not assert deployment for type: %T", v)
	}

	return e, nil
}
