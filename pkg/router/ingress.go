package router

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/provider/k8s"
	ac "k8s.io/api/core/v1"
	ae "k8s.io/api/extensions/v1beta1"
	am "k8s.io/apimachinery/pkg/apis/meta/v1"
	ie "k8s.io/client-go/informers/extensions/v1beta1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type IngressController struct {
	Controller *k8s.Controller
	Router     *Router
}

func NewIngressController(r *Router) (*IngressController, error) {
	ic := &IngressController{
		Router: r,
	}

	c, err := k8s.NewController("convox", "router", ic)
	if err != nil {
		return nil, err
	}

	ic.Controller = c

	return ic, nil
}

func (c *IngressController) Client() kubernetes.Interface {
	return c.Router.Cluster
}

func (c *IngressController) ListOptions(opts *am.ListOptions) {
	opts.LabelSelector = fmt.Sprintf("system=convox")
	opts.ResourceVersion = ""
}

func (c *IngressController) Run() {
	i := ie.NewFilteredIngressInformer(c.Router.Cluster, ac.NamespaceAll, 10*time.Second, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, c.ListOptions)

	ch := make(chan error)

	go c.Controller.Run(i, ch)

	for err := range ch {
		fmt.Printf("err = %+v\n", err)
	}
}

func (c *IngressController) Add(obj interface{}) error {
	i, err := assertIngress(obj)
	if err != nil {
		return err
	}

	for _, r := range i.Spec.Rules {
		c.Router.RackSet(r.Host, i.Labels["rack"])

		host := r.Host

		if !strings.HasSuffix(host, ".") {
			host = fmt.Sprintf("%s.%s.", host, helpers.CoalesceString(i.ObjectMeta.Annotations["convox.domain"], i.ObjectMeta.Labels["rack"]))
		}

		host = strings.TrimSuffix(host, ".")

		for _, port := range r.IngressRuleValue.HTTP.Paths {
			target := rulePathTarget(port, i.ObjectMeta)

			c.Controller.Event(i, ac.EventTypeNormal, "TargetAdd", fmt.Sprintf("%s => %s", host, target))

			c.Router.TargetAdd(host, target)
		}
	}

	if err := c.updateIngressIP(i, "127.0.0.1"); err != nil {
		return err
	}

	return nil
}

func (c *IngressController) Delete(obj interface{}) error {
	i, err := assertIngress(obj)
	if err != nil {
		return err
	}

	for _, r := range i.Spec.Rules {
		for _, port := range r.IngressRuleValue.HTTP.Paths {
			target := rulePathTarget(port, i.ObjectMeta)
			c.Controller.Event(i, ac.EventTypeNormal, "TargetDelete", fmt.Sprintf("%s => %s", r.Host, target))
			c.Router.TargetDelete(r.Host, rulePathTarget(port, i.ObjectMeta))
		}
	}

	return nil
}

func (c *IngressController) Update(prev, cur interface{}) error {
	pi, err := assertIngress(prev)
	if err != nil {
		return err
	}

	ci, err := assertIngress(cur)
	if err != nil {
		return err
	}

	if reflect.DeepEqual(pi.Spec, ci.Spec) {
		return nil
	}

	if err := c.Delete(prev); err != nil {
		return err
	}

	if err := c.Add(cur); err != nil {
		return err
	}

	return nil
}

func (c *IngressController) updateIngressIP(i *ae.Ingress, ip string) error {
	if is := i.Status.LoadBalancer.Ingress; len(is) == 1 && is[0].IP == ip {
		return nil
	}

	i.Status.LoadBalancer.Ingress = []ac.LoadBalancerIngress{
		{IP: ip},
	}

	if _, err := c.Router.Cluster.ExtensionsV1beta1().Ingresses(i.ObjectMeta.Namespace).UpdateStatus(i); err != nil {
		return err
	}

	return nil
}

func assertIngress(v interface{}) (*ae.Ingress, error) {
	i, ok := v.(*ae.Ingress)
	if !ok {
		return nil, fmt.Errorf("could not assert ingress for type: %T", v)
	}

	return i, nil
}

func rulePathTarget(port ae.HTTPIngressPath, meta am.ObjectMeta) string {
	proto := "http"

	if p := meta.Annotations[fmt.Sprintf("convox.ingress.service.%s.%d.protocol", port.Backend.ServiceName, port.Backend.ServicePort.IntVal)]; p != "" {
		proto = p
	}

	return fmt.Sprintf("%s://%s.%s.svc.cluster.local:%d", proto, port.Backend.ServiceName, meta.Namespace, port.Backend.ServicePort.IntVal)
}
