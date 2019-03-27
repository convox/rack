package k8s

import (
	"fmt"
	"os"
	"time"

	ac "k8s.io/api/core/v1"
	am "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	tc "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
)

type Controller struct {
	Handler    ControllerHandler
	Identifier string
	Name       string
	Namespace  string

	errch    chan error
	recorder record.EventRecorder
}

type ControllerHandler interface {
	Add(interface{}) error
	Client() kubernetes.Interface
	Delete(interface{}) error
	Start() error
	Stop() error
	Update(interface{}, interface{}) error
}

func NewController(namespace, name string, handler ControllerHandler) (*Controller, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	c := &Controller{
		Handler:    handler,
		Identifier: hostname,
		Name:       name,
		Namespace:  namespace,
	}

	return c, nil
}

func (c *Controller) Event(object runtime.Object, eventtype, reason, message string) {
	c.recorder.Event(object, eventtype, reason, message)
}

func (c *Controller) Run(informer cache.SharedInformer, ch chan error) {
	c.errch = ch

	eb := record.NewBroadcaster()
	eb.StartRecordingToSink(&tc.EventSinkImpl{Interface: c.Handler.Client().CoreV1().Events("")})

	c.recorder = eb.NewRecorder(scheme.Scheme, ac.EventSource{Component: c.Name})

	rl := &resourcelock.ConfigMapLock{
		ConfigMapMeta: am.ObjectMeta{Namespace: c.Namespace, Name: c.Name},
		Client:        c.Handler.Client().CoreV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity:      c.Identifier,
			EventRecorder: c.recorder,
		},
	}

	el, err := leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
		Lock:          rl,
		LeaseDuration: 15 * time.Second,
		RenewDeadline: 10 * time.Second,
		RetryPeriod:   2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: c.leaderStart(informer),
		},
	})
	if err != nil {
		ch <- err
		return
	}

	go el.Run()
}

func (c *Controller) leaderStart(informer cache.SharedInformer) func(<-chan struct{}) {
	return func(stop <-chan struct{}) {
		fmt.Printf("started leading: %s/%s (%s)\n", c.Namespace, c.Name, c.Identifier)

		informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc:    c.addHandler,
			DeleteFunc: c.deleteHandler,
			UpdateFunc: c.updateHandler,
		})

		if err := c.Handler.Start(); err != nil {
			c.errch <- err
		}

		informer.Run(stop)

		fmt.Printf("stopped leading: %s/%s (%s)\n", c.Namespace, c.Name, c.Identifier)

		if err := c.Handler.Stop(); err != nil {
			c.errch <- err
		}
	}
}

func (c *Controller) addHandler(obj interface{}) {
	if err := c.Handler.Add(obj); err != nil {
		c.errch <- err
	}
}

func (c *Controller) deleteHandler(obj interface{}) {
	if err := c.Handler.Delete(obj); err != nil {
		c.errch <- err
	}
}

func (c *Controller) updateHandler(prev, cur interface{}) {
	// if reflect.DeepEqual(prev, cur) {
	//   return
	// }

	if err := c.Handler.Update(prev, cur); err != nil {
		c.errch <- err
	}
}
