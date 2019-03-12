package k8s

import (
	"fmt"
	"reflect"

	ac "k8s.io/api/core/v1"
	am "k8s.io/apimachinery/pkg/apis/meta/v1"
	ic "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type PodController struct {
	Controller *Controller
	Provider   *Provider
}

func NewPodController(p *Provider) (*PodController, error) {
	pc := &PodController{
		Provider: p,
	}

	c, err := NewController(p.Rack, "convox-k8s-pod", pc)
	if err != nil {
		return nil, err
	}

	pc.Controller = c

	return pc, nil
}

func (c *PodController) Client() kubernetes.Interface {
	return c.Provider.Cluster
}

func (c *PodController) ListOptions(opts *am.ListOptions) {
	opts.LabelSelector = fmt.Sprintf("system=convox,rack=%s", c.Provider.Rack)
	// opts.ResourceVersion = ""
}

func (c *PodController) Run() {
	i := ic.NewFilteredPodInformer(c.Provider.Cluster, ac.NamespaceAll, 0, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, c.ListOptions)

	ch := make(chan error)

	go c.Controller.Run(i, ch)

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

	switch p.Status.Phase {
	case "Succeeded", "Failed":
		if err := c.cleanupPod(p); err != nil {
			return err
		}
	}

	return nil
}

func (c *PodController) Delete(obj interface{}) error {
	return nil
}

func (c *PodController) Update(prev, cur interface{}) error {
	pp, err := assertPod(prev)
	if err != nil {
		return err
	}

	cp, err := assertPod(cur)
	if err != nil {
		return err
	}

	if reflect.DeepEqual(pp.Status, cp.Status) {
		return nil
	}

	// fmt.Printf("pod %s: %s\n", cp.ObjectMeta.Name, cp.Status.Phase)

	switch cp.Status.Phase {
	case "Succeeded", "Failed":
		if err := c.cleanupPod(cp); err != nil {
			return err
		}
	}

	return nil
}

func (c *PodController) cleanupPod(p *ac.Pod) error {
	if err := c.Client().CoreV1().Pods(p.ObjectMeta.Namespace).Delete(p.ObjectMeta.Name, nil); err != nil {
		return err
	}

	return nil
}

func assertPod(v interface{}) (*ac.Pod, error) {
	p, ok := v.(*ac.Pod)
	if !ok {
		return nil, fmt.Errorf("could not assert pod for type: %T", v)
	}

	return p, nil
}
