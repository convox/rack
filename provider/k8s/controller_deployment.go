package k8s

import (
	"fmt"

	ac "k8s.io/api/core/v1"
	ae "k8s.io/api/extensions/v1beta1"
	am "k8s.io/apimachinery/pkg/apis/meta/v1"
	ic "k8s.io/client-go/informers/extensions/v1beta1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type DeploymentController struct {
	Controller *Controller
	Provider   *Provider
}

func NewDeploymentController(p *Provider) (*DeploymentController, error) {
	pc := &DeploymentController{
		Provider: p,
	}

	c, err := NewController(p.Rack, "convox-k8s-deployment", pc)
	if err != nil {
		return nil, err
	}

	pc.Controller = c

	return pc, nil
}

func (c *DeploymentController) Client() kubernetes.Interface {
	return c.Provider.Cluster
}

func (c *DeploymentController) ListOptions(opts *am.ListOptions) {
	// opts.LabelSelector = fmt.Sprintf("system=convox,rack=%s", c.Provider.Rack)
}

func (c *DeploymentController) Run() {
	i := ic.NewFilteredDeploymentInformer(c.Provider.Cluster, ac.NamespaceAll, 0, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, c.ListOptions)

	ch := make(chan error)

	go c.Controller.Run(i, ch)

	for err := range ch {
		fmt.Printf("err = %+v\n", err)
	}
}

func (c *DeploymentController) Start() error {
	return nil
}

func (c *DeploymentController) Stop() error {
	return nil
}

func (c *DeploymentController) Add(obj interface{}) error {
	// d, err := assertDeployment(obj)
	// if err != nil {
	//   return err
	// }

	// fmt.Printf("deployment add: %s/%s\n", d.ObjectMeta.Namespace, d.ObjectMeta.Name)

	return nil
}

func (c *DeploymentController) Delete(obj interface{}) error {
	// d, err := assertDeployment(obj)
	// if err != nil {
	//   return err
	// }

	// fmt.Printf("deployment delete: %s/%s\n", d.ObjectMeta.Namespace, d.ObjectMeta.Name)

	return nil
}

func (c *DeploymentController) Update(prev, cur interface{}) error {
	pd, err := assertDeployment(prev)
	if err != nil {
		return err
	}

	cd, err := assertDeployment(cur)
	if err != nil {
		return err
	}

	if pd.ResourceVersion == cd.ResourceVersion {
		return nil
	}

	fmt.Printf("deployment update: %s/%s\n", cd.ObjectMeta.Namespace, cd.ObjectMeta.Name)

	sc, rc := deploymentCondition(cd, "Progressing")
	sp, _ := deploymentCondition(pd, "Progressing")

	if sc == "False" && sp == "True" && rc == "ProgressDeadlineExceeded" {
		fmt.Printf("rollback: %s/%s\n", cd.ObjectMeta.Namespace, cd.ObjectMeta.Name)
	}

	// if deploymentCondition(cd, "

	return nil
}

func assertDeployment(v interface{}) (*ae.Deployment, error) {
	d, ok := v.(*ae.Deployment)
	if !ok {
		return nil, fmt.Errorf("could not assert deployment for type: %T", v)
	}

	return d, nil
}

func deploymentCondition(d *ae.Deployment, name string) (string, string) {
	for _, c := range d.Status.Conditions {
		if string(c.Type) == name {
			return string(c.Status), c.Reason
		}
	}

	return "", ""
}
