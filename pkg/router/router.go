package router

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"sync"
	"time"

	ae "k8s.io/api/extensions/v1beta1"
	am "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	idleCheck   = 1 * time.Minute
	idleTimeout = 60 * time.Minute
)

var (
	activityLock sync.Mutex
	idleLock     sync.Mutex
	targetLock   sync.Mutex
)

type Router struct {
	Cluster  kubernetes.Interface
	DNS      Server
	HTTP     Server
	HTTPS    Server
	IP       string
	activity map[string]time.Time
	active   map[string]int
	idle     map[string]bool
	routes   map[string]map[string]bool
	racks    map[string]string
}

type Server interface {
	ListenAndServe() error
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func New() (*Router, error) {
	r := &Router{
		activity: map[string]time.Time{},
		active:   map[string]int{},
		idle:     map[string]bool{},
		routes:   map[string]map[string]bool{},
		racks:    map[string]string{},
	}

	c, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	kc, err := kubernetes.NewForConfig(c)
	if err != nil {
		return nil, err
	}

	dns, err := NewDNS(r)
	if err != nil {
		return nil, err
	}

	// http, err := NewHTTP("http", 80, r)
	// if err != nil {
	//   return nil, err
	// }

	// http.Handler = redirectHTTPS(http.ServeRequest)

	https, err := NewHTTP(443, r)
	if err != nil {
		return nil, err
	}

	http := &http.Server{Addr: ":80", Handler: redirectHTTPS(https.ServeRequest)}

	r.Cluster = kc
	r.DNS = dns
	r.HTTP = http
	r.HTTPS = https

	ic, err := NewIngressController(r)
	if err != nil {
		return nil, err
	}

	s, err := kc.CoreV1().Services("convox-system").Get("router", am.GetOptions{})
	if err != nil {
		return nil, err
	}

	if len(s.Status.LoadBalancer.Ingress) > 0 && s.Status.LoadBalancer.Ingress[0].Hostname == "localhost" {
		r.IP = "127.0.0.1"
	} else {
		r.IP = s.Spec.ClusterIP
	}

	go ic.Run()

	return r, nil
}

func (r *Router) ActivityBegin(host string) {
	activityLock.Lock()
	defer activityLock.Unlock()

	r.activity[host] = time.Now().UTC()
	r.active[host] += 1
}

func (r *Router) ActivityEnd(host string) {
	activityLock.Lock()
	defer activityLock.Unlock()

	r.active[host] -= 1
}

func (r *Router) ActivityGet(host string) (time.Time, int) {
	activityLock.Lock()
	defer activityLock.Unlock()

	return r.activity[host], r.active[host]
}

func (r *Router) ActivityOld() []string {
	activityLock.Lock()
	defer activityLock.Unlock()

	hs := []string{}

	for host, activity := range r.activity {
		if activity.Before(time.Now().UTC().Add(-1*idleTimeout)) && r.active[host] == 0 {
			hs = append(hs, host)
		}
	}

	return hs
}

func (r *Router) HostBegin(host string) {
	r.ActivityBegin(host)
	r.HostUnidle(host)
}

func (r *Router) HostEnd(host string) {
	r.ActivityEnd(host)
}

func (r *Router) HostIdleGet(host string) bool {
	idleLock.Lock()
	defer idleLock.Unlock()

	return r.idle[host]
}

func (r *Router) HostIdleSet(host string, idle bool) {
	idleLock.Lock()
	defer idleLock.Unlock()

	r.idle[host] = idle
}

func (r *Router) HostIdle(host string) {
	if r.HostIdleGet(host) {
		return
	}

	fmt.Printf("ns=convox.router at=idle host=%q\n", host)

	r.HostIdleSet(host, true)

	for target := range r.routes[host] {
		if service, namespace, ok := parseTarget(target); ok {
			scale := &ae.Scale{
				ObjectMeta: am.ObjectMeta{
					Namespace: namespace,
					Name:      service,
				},
				Spec: ae.ScaleSpec{Replicas: 0},
			}

			if _, err := r.Cluster.ExtensionsV1beta1().Deployments(namespace).UpdateScale(service, scale); err != nil {
				fmt.Printf("ns=convox.router at=idle host=%q error=%q\n", host, err)
			}
		}
	}
}

func (r *Router) HostUnidle(host string) {
	if !r.HostIdleGet(host) {
		return
	}

	fmt.Printf("ns=convox.router at=unidle host=%q state=unidling\n", host)

	r.HostIdleSet(host, false)

	for target := range r.routes[host] {
		if service, namespace, ok := parseTarget(target); ok {
			scale := &ae.Scale{
				ObjectMeta: am.ObjectMeta{
					Namespace: namespace,
					Name:      service,
				},
				Spec: ae.ScaleSpec{Replicas: 1},
			}

			if _, err := r.Cluster.ExtensionsV1beta1().Deployments(namespace).UpdateScale(service, scale); err != nil {
				fmt.Printf("ns=convox.router at=unidle host=%q error=%q\n", host, err)
			}

			for {
				time.Sleep(200 * time.Millisecond)
				if rs, err := r.Cluster.AppsV1().Deployments(namespace).Get(service, am.GetOptions{}); err == nil {
					if rs.Status.ReadyReplicas > 0 {
						time.Sleep(500 * time.Millisecond)
						break
					}
				}
			}
		}
	}

	fmt.Printf("ns=convox.router at=unidle host=%q state=ready\n", host)
}

func (r *Router) Serve() error {
	ch := make(chan error, 1)

	go serve(ch, r.DNS)
	go serve(ch, r.HTTP)
	go serve(ch, r.HTTPS)

	go r.idleTicker()

	return <-ch
}

func (r *Router) RackSet(host, rack string) {
	r.racks[host] = rack
}

func (r *Router) Route(host string) (string, error) {
	targetLock.Lock()
	defer targetLock.Unlock()

	if r.routes[host] == nil {
		return "", fmt.Errorf("unknown host")
	}

	if len(r.routes[host]) == 0 {
		return "", fmt.Errorf("no backends available")
	}

	r.HostBegin(host)
	defer r.HostEnd(host)

	targets := []string{}

	for target := range r.routes[host] {
		targets = append(targets, target)
	}

	return targets[rand.Intn(len(targets))], nil
}

func (r *Router) TargetAdd(host, target string) {
	targetLock.Lock()
	defer targetLock.Unlock()

	fmt.Printf("ns=convox.router at=target.add host=%q target=%q\n", host, target)

	if r.routes[host] == nil {
		r.routes[host] = map[string]bool{}
	}

	r.routes[host][target] = true

	if service, namespace, ok := parseTarget(target); ok {
		if rs, err := r.Cluster.AppsV1().Deployments(namespace).Get(service, am.GetOptions{}); err == nil {
			if rs.Status.Replicas == 0 {
				r.HostIdle(host)
			}
		}
	}
}

func (r *Router) TargetCount(host string) int {
	targetLock.Lock()
	defer targetLock.Unlock()

	targets, ok := r.routes[host]
	if !ok {
		return 0
	}

	return len(targets)
}

func (r *Router) TargetDelete(host, target string) {
	targetLock.Lock()
	defer targetLock.Unlock()

	fmt.Printf("ns=convox.router at=target.delete host=%q target=%q\n", host, target)

	if r.routes[host] != nil {
		delete(r.routes[host], target)
	}
}

func (r *Router) idleTicker() {
	for range time.Tick(idleCheck) {
		if err := r.idleTick(); err != nil {
			fmt.Printf("ns=convox.router at=idle.ticker error=%v\n", err)
		}
	}
}

func (r *Router) idleTick() error {
	targetLock.Lock()
	defer targetLock.Unlock()

	for _, host := range r.ActivityOld() {
		activity, active := r.ActivityGet(host)
		age := activity.Sub(time.Now().UTC()).Truncate(time.Second) * -1
		fmt.Printf("ns=convox.router at=idle.tick host=%q age=%s active=%d idle=%t\n", host, age, active, r.HostIdleGet(host))
		r.HostIdle(host)
	}

	return nil
}

var reTarget = regexp.MustCompile(`^([^.]+)\.([^.]+)\.svc\.cluster\.local$`)

func parseTarget(target string) (string, string, bool) {
	u, err := url.Parse(target)
	if err != nil {
		return "", "", false
	}

	if m := reTarget.FindStringSubmatch(u.Hostname()); len(m) == 3 {
		return m[1], m[2], true
	}

	return "", "", false
}

func redirectHTTPS(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Forwarded-Proto") == "https" {
			fn(w, r)
			return
		}

		target := url.URL{Scheme: "https", Host: r.Host, Path: r.URL.Path, RawQuery: r.URL.RawQuery}

		http.Redirect(w, r, target.String(), http.StatusMovedPermanently)
	}
}

func serve(ch chan error, s Server) {
	ch <- s.ListenAndServe()
}
