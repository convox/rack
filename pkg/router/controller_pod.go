package router

import (
	"fmt"

	"github.com/convox/rack/pkg/kctl"
	ac "k8s.io/api/core/v1"
	am "k8s.io/apimachinery/pkg/apis/meta/v1"
	ic "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type PodController struct {
	controller *kctl.Controller
	kc         kubernetes.Interface
	router     BackendRouter
}

func NewPodController(kc kubernetes.Interface, router BackendRouter) (*PodController, error) {
	ic := &PodController{kc: kc, router: router}

	c, err := kctl.NewController("convox-system", "convox-router-pod", ic)
	if err != nil {
		return nil, err
	}

	ic.controller = c

	return ic, nil
}

func (c *PodController) Client() kubernetes.Interface {
	return c.kc
}

func (c *PodController) ListOptions(opts *am.ListOptions) {
	opts.LabelSelector = fmt.Sprintf("system=convox")
	opts.ResourceVersion = ""
}

func (c *PodController) Run() {
	i := ic.NewFilteredPodInformer(c.kc, ac.NamespaceAll, 0, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, c.ListOptions)

	ch := make(chan error)

	go c.controller.Run(i, ch)

	for err := range ch {
		fmt.Printf("err = %+v\n", err)
	}
}

func (c *PodController) Start() error {
	return nil
}

func (c *PodController) Stop() error {
	return nil
}

func (c *PodController) Add(obj interface{}) error {
	p, err := assertPod(obj)
	if err != nil {
		return err
	}

	fmt.Printf("ns=controller.pod at=add pod=%s\n", p.ObjectMeta.Name)

	is, err := c.kc.ExtensionsV1beta1().Ingresses(p.ObjectMeta.Namespace).List(am.ListOptions{})
	if err != nil {
		return err
	}

	// change a host's state to non-idle when one of its pods starts
	for _, i := range is.Items {
		for _, r := range i.Spec.Rules {
			for _, port := range r.IngressRuleValue.HTTP.Paths {
				target := rulePathTarget(port, i.ObjectMeta)
				c.router.IdleSet(target, false)
			}
		}
	}

	return nil
}

func (c *PodController) Delete(obj interface{}) error {
	return nil
}

func (c *PodController) Update(prev, cur interface{}) error {
	return nil
}

func assertPod(v interface{}) (*ac.Pod, error) {
	p, ok := v.(*ac.Pod)
	if !ok {
		return nil, fmt.Errorf("could not assert pod for type: %T", v)
	}

	return p, nil
}
