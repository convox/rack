package k8s

import (
	"bufio"
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/convox/rack/pkg/kctl"
	ac "k8s.io/api/core/v1"
	am "k8s.io/apimachinery/pkg/apis/meta/v1"
	ic "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type PodController struct {
	Controller *kctl.Controller
	Provider   *Provider

	logger *podLogger
	start  time.Time
}

func NewPodController(p *Provider) (*PodController, error) {
	pc := &PodController{
		Provider: p,
		logger:   NewPodLogger(p),
		start:    time.Now().UTC(),
	}

	c, err := kctl.NewController(p.Rack, "convox-k8s-pod", pc)
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
	c.start = time.Now().UTC()

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

	// fmt.Printf("pod add %s/%s: %s\n", p.ObjectMeta.Namespace, p.ObjectMeta.Name, p.Status.Phase)

	switch p.Status.Phase {
	case "Succeeded", "Failed":
		go c.cleanupPod(p)
	case "Pending", "Running":
		c.logger.Start(p.ObjectMeta.Namespace, p.ObjectMeta.Name, c.start)
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

	// fmt.Printf("pod update %s/%s: %s (was %s)\n", cp.ObjectMeta.Namespace, cp.ObjectMeta.Name, cp.Status.Phase, pp.Status.Phase)

	if cp.Status.Phase != pp.Status.Phase {
		switch cp.Status.Phase {
		case "Succeeded", "Failed":
			go c.cleanupPod(cp)
		}
	}

	return nil
}

func (c *PodController) cleanupPod(p *ac.Pod) error {
	time.Sleep(5 * time.Second)

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

// func podCondition(p *ac.Pod, name string) *ac.PodCondition {
//   for _, c := range p.Status.Conditions {
//     if string(c.Type) == name {
//       return &c
//     }
//   }

//   return nil
// }

type podLogger struct {
	provider *Provider
	streams  sync.Map
}

func NewPodLogger(p *Provider) *podLogger {
	return &podLogger{provider: p}
}

func (l *podLogger) Start(namespace, pod string, start time.Time) {
	key := fmt.Sprintf("%s:%s", namespace, pod)

	ctx, cancel := context.WithCancel(context.Background())

	if _, exists := l.streams.LoadOrStore(key, cancel); !exists {
		go l.watch(ctx, namespace, pod, start)
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

func (l *podLogger) stream(ch chan string, namespace, pod string, start time.Time) {
	defer close(ch)

	since := am.NewTime(start)

	for {
		lopts := &ac.PodLogOptions{
			Follow:     true,
			SinceTime:  &since,
			Timestamps: true,
		}
		r, err := l.provider.Cluster.CoreV1().Pods(namespace).GetLogs(pod, lopts).Stream()
		if err != nil {
			fmt.Printf("err = %+v\n", err)
			break
		}

		s := bufio.NewScanner(r)

		s.Buffer(make([]byte, ScannerStartSize), ScannerMaxSize)

		for s.Scan() {
			line := s.Text()

			if ts, err := time.Parse(time.RFC3339Nano, strings.Split(line, " ")[0]); err == nil {
				since = am.NewTime(ts)
			}

			ch <- line
		}

		if err := s.Err(); err != nil {
			fmt.Printf("err = %+v\n", err)
			continue
		}

		break
	}
}

func (l *podLogger) watch(ctx context.Context, namespace, pod string, start time.Time) {
	defer l.Stop(namespace, pod)

	ch := make(chan string)

	var p *ac.Pod
	var err error

	for {
		p, err = l.provider.Cluster.CoreV1().Pods(namespace).Get(pod, am.GetOptions{})
		if err != nil {
			fmt.Printf("err = %+v\n", err)
			return
		}

		if p.Status.Phase != "Pending" {
			break
		}

		time.Sleep(1 * time.Second)
	}

	app := p.ObjectMeta.Labels["app"]
	typ := p.ObjectMeta.Labels["type"]
	name := p.ObjectMeta.Labels["name"]

	if typ == "process" {
		typ = "service"
	}

	go l.stream(ch, namespace, pod, start)

	for {
		select {
		case <-ctx.Done():
			return
		case log, ok := <-ch:
			if !ok {
				return
			}
			if parts := strings.SplitN(log, " ", 2); len(parts) == 2 {
				if ts, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
					l.provider.Engine.Log(app, fmt.Sprintf("%s/%s/%s", typ, name, pod), ts, parts[1])
				}
			}
		}
	}
}
