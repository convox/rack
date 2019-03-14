package k8s

import (
	"bufio"
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/convox/rack/pkg/options"
	ac "k8s.io/api/core/v1"
	am "k8s.io/apimachinery/pkg/apis/meta/v1"
	ic "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type PodController struct {
	Controller *Controller
	Provider   *Provider

	logger *podLogger
}

func NewPodController(p *Provider) (*PodController, error) {
	pc := &PodController{
		Provider: p,
		logger:   NewPodLogger(p),
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
	case "Running":
		c.logger.Start(p.ObjectMeta.Namespace, p.ObjectMeta.Name)
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
	case "Running":
		c.logger.Start(cp.ObjectMeta.Namespace, cp.ObjectMeta.Name)
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

// const (
//   ScannerStartSize = 4096
//   ScannerMaxSize   = 1024 * 1024
// )

type podLogger struct {
	provider *Provider
	streams  sync.Map
}

func NewPodLogger(p *Provider) *podLogger {
	return &podLogger{provider: p}
}

func (l *podLogger) Start(namespace, pod string) {
	key := fmt.Sprintf("%s:%s", namespace, pod)

	ctx, cancel := context.WithCancel(context.Background())

	if _, exists := l.streams.LoadOrStore(key, cancel); !exists {
		go l.watch(ctx, namespace, pod)
	}
}

func (l *podLogger) Stop(namespace, pod string) {
	key := fmt.Sprintf("%s:%s", namespace, pod)

	if cv, ok := l.streams.Load(key); ok {
		if cfn, ok := cv.(context.CancelFunc); ok {
			cfn()
		}
		l.streams.Delete(key)
	}
}

func (l *podLogger) stream(ch chan string, namespace, pod string) {
	for {
		lopts := &ac.PodLogOptions{
			Follow:       true,
			SinceSeconds: options.Int64(1),
			Timestamps:   true,
		}
		r, err := l.provider.Cluster.CoreV1().Pods(namespace).GetLogs(pod, lopts).Stream()
		if err != nil {
			close(ch)
			return
		}

		s := bufio.NewScanner(r)

		s.Buffer(make([]byte, ScannerStartSize), ScannerMaxSize)

		for s.Scan() {
			ch <- s.Text()
		}

		if err := s.Err(); err != nil {
			fmt.Printf("err = %+v\n", err)
		}
	}
}

func (l *podLogger) watch(ctx context.Context, namespace, pod string) {
	defer l.Stop(namespace, pod)

	ch := make(chan string)

	p, err := l.provider.Cluster.CoreV1().Pods(namespace).Get(pod, am.GetOptions{})
	if err != nil {
		fmt.Printf("err = %+v\n", err)
		return
	}

	app := p.ObjectMeta.Labels["app"]

	go l.stream(ch, namespace, pod)

	for {
		select {
		case <-ctx.Done():
			return
		case log, ok := <-ch:
			if parts := strings.SplitN(log, " ", 2); len(parts) == 2 {
				if ts, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
					l.provider.Engine.Log(app, pod, ts, parts[1])
				}
			}

			if !ok {
				fmt.Println("stream closed")
				return
			}
		}
	}
}
