package router

import (
	"fmt"
	"time"

	"github.com/convox/rack/pkg/kctl"
	ac "k8s.io/api/core/v1"
	ae "k8s.io/api/extensions/v1beta1"
	am "k8s.io/apimachinery/pkg/apis/meta/v1"
	ie "k8s.io/client-go/informers/extensions/v1beta1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type DeploymentController struct {
	backend    *BackendKubernetes
	controller *kctl.Controller
	kc         kubernetes.Interface
}

func NewDeploymentController(kc kubernetes.Interface, backend *BackendKubernetes) (*DeploymentController, error) {
	ic := &DeploymentController{backend: backend, kc: kc}

	c, err := kctl.NewController("convox-system", "convox-router-deployment", ic)
	if err != nil {
		return nil, err
	}

	ic.controller = c

	return ic, nil
}

func (c *DeploymentController) Client() kubernetes.Interface {
	return c.kc
}

func (c *DeploymentController) ListOptions(opts *am.ListOptions) {
	opts.LabelSelector = fmt.Sprintf("system=convox")
	opts.ResourceVersion = ""
}

func (c *DeploymentController) Run() {
	i := ie.NewFilteredDeploymentInformer(c.kc, ac.NamespaceAll, 5*time.Minute, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, c.ListOptions)

	ch := make(chan error)

	go c.controller.Run(i, ch)

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
	d, err := assertDeployment(obj)
	if err != nil {
		return err
	}

	fmt.Printf("ns=controller.deployment at=add deployment=%s\n", d.ObjectMeta.Name)

	if err := c.syncDeployment(d); err != nil {
		return err
	}

	return nil
}

func (c *DeploymentController) Delete(obj interface{}) error {
	d, err := assertDeployment(obj)
	if err != nil {
		return err
	}

	fmt.Printf("ns=controller.deployment at=delete deployment=%s\n", d.ObjectMeta.Name)

	return nil
}

func (c *DeploymentController) Update(prev, cur interface{}) error {
	cd, err := assertDeployment(cur)
	if err != nil {
		return err
	}

	fmt.Printf("ns=controller.deployment at=update deployment=%s\n", cd.ObjectMeta.Name)

	if err := c.syncDeployment(cd); err != nil {
		return err
	}

	return nil
}

func (c *DeploymentController) syncDeployment(d *ae.Deployment) error {
	if d.Spec.Replicas == nil {
		return c.backend.IdleUpdate(d.Namespace, d.Name, true)
	}

	return c.backend.IdleUpdate(d.Namespace, d.Name, *d.Spec.Replicas == 0)
}

func assertDeployment(v interface{}) (*ae.Deployment, error) {
	d, ok := v.(*ae.Deployment)
	if !ok {
		return nil, fmt.Errorf("could not assert deployment for type: %T", v)
	}

	return d, nil
}
