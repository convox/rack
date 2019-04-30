package k8s

import (
	"fmt"

	"github.com/convox/rack/pkg/kctl"
	"github.com/convox/rack/pkg/options"
	ac "k8s.io/api/core/v1"
	am "k8s.io/apimachinery/pkg/apis/meta/v1"
	ic "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type NodeController struct {
	Controller *kctl.Controller
	Provider   *Provider
}

func NewNodeController(p *Provider) (*NodeController, error) {
	pc := &NodeController{
		Provider: p,
	}

	c, err := kctl.NewController(p.Rack, "convox-k8s-node", pc)
	if err != nil {
		return nil, err
	}

	pc.Controller = c

	return pc, nil
}

func (c *NodeController) Client() kubernetes.Interface {
	return c.Provider.Cluster
}

func (c *NodeController) ListOptions(opts *am.ListOptions) {
	// opts.LabelSelector = fmt.Sprintf("system=convox,rack=%s", c.Provider.Rack)
}

func (c *NodeController) Run() {
	i := ic.NewFilteredNodeInformer(c.Provider.Cluster, 0, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, c.ListOptions)

	ch := make(chan error)

	go c.Controller.Run(i, ch)

	for err := range ch {
		fmt.Printf("err = %+v\n", err)
	}
}

func (c *NodeController) Start() error {
	return nil
}

func (c *NodeController) Stop() error {
	return nil
}

func (c *NodeController) Add(obj interface{}) error {
	n, err := assertNode(obj)
	if err != nil {
		return err
	}

	fmt.Printf("node add: %s\n", n.ObjectMeta.Name)

	if err := c.updateRouterScale(); err != nil {
		return err
	}

	return nil
}

func (c *NodeController) Delete(obj interface{}) error {
	n, err := assertNode(obj)
	if err != nil {
		return err
	}

	fmt.Printf("node delete: %s\n", n.ObjectMeta.Name)

	if err := c.updateRouterScale(); err != nil {
		return err
	}

	return nil
}

func (c *NodeController) Update(prev, cur interface{}) error {
	return nil
}

func (c *NodeController) updateRouterScale() error {
	ns, err := c.Provider.Cluster.CoreV1().Nodes().List(am.ListOptions{})
	if err != nil {
		return err
	}

	min := len(ns.Items)
	max := min

	as, err := c.Provider.Cluster.AutoscalingV1().HorizontalPodAutoscalers("convox-system").Get("router", am.GetOptions{})
	if err != nil {
		return err
	}

	as.Spec.MinReplicas = options.Int32(1)
	as.Spec.MaxReplicas = int32(max)

	if _, err := c.Provider.Cluster.AutoscalingV1().HorizontalPodAutoscalers("convox-system").Update(as); err != nil {
		return err
	}

	return nil
}

func assertNode(v interface{}) (*ac.Node, error) {
	p, ok := v.(*ac.Node)
	if !ok {
		return nil, fmt.Errorf("could not assert pod for type: %T", v)
	}

	return p, nil
}
