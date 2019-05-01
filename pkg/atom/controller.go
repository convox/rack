package atom

import (
	"fmt"
	"time"

	ct "github.com/convox/rack/pkg/atom/pkg/apis/convox/v1"
	cv "github.com/convox/rack/pkg/atom/pkg/client/clientset/versioned"
	ic "github.com/convox/rack/pkg/atom/pkg/client/informers/externalversions/convox/v1"
	"github.com/convox/rack/pkg/kctl"
	"github.com/pkg/errors"
	ac "k8s.io/api/core/v1"
	am "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type AtomController struct {
	atom       *Client
	controller *kctl.Controller
	convox     cv.Interface
	kubernetes kubernetes.Interface
}

func NewController(cfg *rest.Config) (*AtomController, error) {
	ac, err := New(cfg)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	cc, err := cv.NewForConfig(cfg)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	kc, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	acc := &AtomController{
		atom:       ac,
		convox:     cc,
		kubernetes: kc,
	}

	c, err := kctl.NewController("kube-system", "convox-atom", acc)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	acc.controller = c

	return acc, nil
}

func (c *AtomController) Client() kubernetes.Interface {
	return c.kubernetes
}

func (c *AtomController) ListOptions(opts *am.ListOptions) {
}

func (c *AtomController) Run() {
	i := ic.NewFilteredAtomInformer(c.convox, ac.NamespaceAll, 5*time.Second, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, c.ListOptions)

	ch := make(chan error)

	go c.controller.Run(i, ch)

	for err := range ch {
		fmt.Printf("err = %+v\n", err)
	}
}

func (c *AtomController) Start() error {
	return nil
}

func (c *AtomController) Stop() error {
	return nil
}

func (c *AtomController) Add(obj interface{}) error {
	return nil
}

func (c *AtomController) Delete(obj interface{}) error {
	return nil
}

func (c *AtomController) Update(prev, cur interface{}) error {
	pa, err := assertAtom(prev)
	if err != nil {
		return errors.WithStack(err)
	}

	ca, err := assertAtom(cur)
	if err != nil {
		return errors.WithStack(err)
	}

	switch ca.Status {
	case "Failed", "Reverted", "Success":
		if pa.ResourceVersion == ca.ResourceVersion {
			return nil
		}
	}

	fmt.Printf("atom: %s/%s (%s)\n", ca.Namespace, ca.Name, ca.Status)

	// if ca.Spec.Current != pa.Spec.Current {
	//   fmt.Printf("atom changed: %s/%s\n", ca.Namespace, ca.Name)

	//   return nil
	// }

	switch ca.Status {
	case "Cancelled", "Deadline", "Error":
		if err := c.atom.rollback(ca); err != nil {
			c.atom.status(ca, "Failed")
			return errors.WithStack(err)
		}
	case "Pending":
		if err := c.atom.apply(ca); err != nil {
			c.atom.status(ca, "Rollback")
			return errors.WithStack(err)
		}
	case "Rollback":
		if deadline := am.NewTime(time.Now().UTC().Add(-1 * time.Duration(ca.Spec.ProgressDeadlineSeconds) * time.Second)); ca.Started.Before(&deadline) {
			c.atom.status(ca, "Failed")
			return nil
		}

		success, err := c.atom.check(ca)
		if err != nil {
			c.atom.status(ca, "Failed")
			return errors.WithStack(err)
		}

		if success {
			c.atom.status(ca, "Reverted")
		}
	case "Running":
		if deadline := am.NewTime(time.Now().UTC().Add(-1 * time.Duration(ca.Spec.ProgressDeadlineSeconds) * time.Second)); ca.Started.Before(&deadline) {
			c.atom.status(ca, "Deadline")
			return nil
		}

		success, err := c.atom.check(ca)
		if err != nil {
			c.atom.status(ca, "Error")
			return errors.WithStack(err)
		}

		if success {
			c.atom.status(ca, "Success")
		}
	}

	return nil
}

func assertAtom(v interface{}) (*ct.Atom, error) {
	a, ok := v.(*ct.Atom)
	if !ok {
		return nil, errors.WithStack(fmt.Errorf("could not assert atom for type: %T", v))
	}

	return a, nil
}
