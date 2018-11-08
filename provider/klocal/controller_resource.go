package klocal

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"reflect"
	"strings"
	"time"

	"github.com/convox/logger"
	"github.com/convox/rack/provider/k8s"
	acx "github.com/convox/rack/provider/k8s/pkg/apis/convox/v1"
	"github.com/convox/rack/provider/k8s/pkg/client/clientset/versioned"
	icx "github.com/convox/rack/provider/k8s/pkg/client/informers/externalversions/convox/v1"
	ac "k8s.io/api/core/v1"
	am "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type ResourceController struct {
	Controller *k8s.Controller
	Provider   *k8s.Provider

	client versioned.Interface
	logger *logger.Logger
}

func NewResourceController(p *k8s.Provider) (*ResourceController, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	cl, err := versioned.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	rc := &ResourceController{
		Provider: p,
		client:   cl,
		logger:   logger.New("ns=controller.resource"),
	}

	c, err := k8s.NewController(p.Rack, "convox-resource", rc)
	if err != nil {
		return nil, err
	}

	rc.Controller = c

	return rc, nil
}

func (c *ResourceController) Client() kubernetes.Interface {
	return c.Provider.Cluster
}

func (c *ResourceController) ListOptions(opts *am.ListOptions) {
	opts.LabelSelector = fmt.Sprintf("system=convox,rack=%s", c.Provider.Rack)
	// opts.ResourceVersion = ""
}

func (c *ResourceController) Run() {
	i := icx.NewFilteredExternalResourceInformer(c.client, ac.NamespaceAll, 10*time.Second, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, c.ListOptions)

	ch := make(chan error)

	go c.Controller.Run(i, ch)

	for err := range ch {
		fmt.Printf("err = %+v\n", err)
	}
}

func (c *ResourceController) Add(obj interface{}) error {
	log := c.logger.At("Add")

	r, err := assertResource(obj)
	if err != nil {
		return log.Error(err)
	}

	log = log.Append("app=%s resource=%s type=%s", r.ObjectMeta.Labels["app"], r.Name, r.Spec.Type)

	tdata, err := c.resourceTemplate(r)
	if err != nil {
		return err
	}

	// fmt.Printf("string(tdata) = %+v\n", string(tdata))

	cmd := exec.Command("kubectl", "apply", "--prune", "-l", fmt.Sprintf("system=convox,scope=resource,rack=%s,app=%s,resource=%s", r.ObjectMeta.Labels["rack"], r.ObjectMeta.Labels["app"], r.Name), "-f", "-")

	cmd.Stdin = bytes.NewReader(tdata)

	data, err := cmd.CombinedOutput()
	if err != nil {
		return log.Error(errors.New(strings.TrimSpace(string(data))))
	}

	// fmt.Printf("string(data) = %+v\n", string(data))

	return log.Success()
}

func (c *ResourceController) Delete(obj interface{}) error {
	log := c.logger.At("Delete")

	r, err := assertResource(obj)
	if err != nil {
		return log.Error(err)
	}

	log = log.Append("app=%s resource=%s type=%s", r.ObjectMeta.Labels["app"], r.Name, r.Spec.Type)

	tdata, err := c.resourceTemplate(r)
	if err != nil {
		return err
	}

	// fmt.Printf("string(tdata) = %+v\n", string(tdata))

	cmd := exec.Command("kubectl", "delete", "-f", "-")

	cmd.Stdin = bytes.NewReader(tdata)

	data, err := cmd.CombinedOutput()
	if err != nil {
		return log.Error(errors.New(strings.TrimSpace(string(data))))
	}

	// fmt.Printf("string(data) = %+v\n", string(data))

	return log.Success()
}

func (c *ResourceController) Update(prev, cur interface{}) error {
	log := c.logger.At("Update")

	pr, err := assertResource(prev)
	if err != nil {
		return log.Error(err)
	}

	cr, err := assertResource(cur)
	if err != nil {
		return log.Error(err)
	}

	if reflect.DeepEqual(pr.Spec, cr.Spec) {
		return nil
	}

	log = log.Append("app=%s resource=%s type=%s", cr.ObjectMeta.Labels["app"], cr.Name, cr.Spec.Type)

	tdata, err := c.resourceTemplate(cr)
	if err != nil {
		return err
	}

	// fmt.Printf("string(tdata) = %+v\n", string(tdata))

	cmd := exec.Command("kubectl", "apply", "--prune", "-l", fmt.Sprintf("system=convox,scope=resource,rack=%s,app=%s,resource=%s", cr.ObjectMeta.Labels["rack"], cr.ObjectMeta.Labels["app"], cr.Name), "-f", "-")

	cmd.Stdin = bytes.NewReader(tdata)

	data, err := cmd.CombinedOutput()
	if err != nil {
		return log.Error(errors.New(strings.TrimSpace(string(data))))
	}

	// fmt.Printf("string(data) = %+v\n", string(data))

	return log.Success()
}

func (c *ResourceController) resourceTemplate(r *acx.ExternalResource) ([]byte, error) {
	params := map[string]interface{}{
		"App":        r.ObjectMeta.Labels["app"],
		"Namespace":  r.ObjectMeta.Namespace,
		"Name":       r.Name,
		"Parameters": r.Spec.Parameters,
		"Password":   "foo",
		"Rack":       r.ObjectMeta.Labels["rack"],
	}

	return c.Provider.RenderTemplate("klocal", "resource/postgres", params)
}

func assertResource(v interface{}) (*acx.ExternalResource, error) {
	r, ok := v.(*acx.ExternalResource)
	if !ok {
		return nil, fmt.Errorf("could not assert resource for type: %T", v)
	}

	return r, nil
}
