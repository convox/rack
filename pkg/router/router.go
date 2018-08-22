package router

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Router struct {
	Cluster kubernetes.Interface
	DNS     *DNS
	HTTP    *HTTP
	HTTPS   *HTTP
	routes  map[string]map[string]bool
	racks   map[string]string
}

type Server interface {
	Serve() error
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func New() (*Router, error) {
	r := &Router{
		routes: map[string]map[string]bool{},
		racks:  map[string]string{},
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

	http, err := NewHTTP(r, "http", 80)
	if err != nil {
		return nil, err
	}

	http.Handler = redirectHTTPS

	https, err := NewHTTP(r, "https", 443)
	if err != nil {
		return nil, err
	}

	r.Cluster = kc
	r.DNS = dns
	r.HTTP = http
	r.HTTPS = https

	ic, err := NewIngressController(r)
	if err != nil {
		return nil, err
	}

	go ic.Run()

	return r, nil
}

func (r *Router) Serve() error {
	ch := make(chan error, 1)

	go serve(ch, r.DNS)
	go serve(ch, r.HTTP)
	go serve(ch, r.HTTPS)

	return <-ch
}

func (r *Router) RackSet(host, rack string) {
	r.racks[host] = rack
}

var targetLock sync.Mutex

func (r *Router) TargetAdd(host, target string) {
	targetLock.Lock()
	defer targetLock.Unlock()

	fmt.Printf("ns=convox.router at=target.add host=%q target=%q\n", host, target)

	if r.routes[host] == nil {
		r.routes[host] = map[string]bool{}
	}

	r.routes[host][target] = true
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

func (r *Router) TargetRandom(host string) string {
	targetLock.Lock()
	defer targetLock.Unlock()

	if r.routes[host] == nil || len(r.routes[host]) == 0 {
		return ""
	}

	targets := []string{}

	for target := range r.routes[host] {
		targets = append(targets, target)
	}

	return targets[rand.Intn(len(targets))]
}

func redirectHTTPS(w http.ResponseWriter, r *http.Request) {
	target := url.URL{Scheme: "https", Host: r.Host, Path: r.URL.Path, RawQuery: r.URL.RawQuery}

	http.Redirect(w, r, target.String(), http.StatusMovedPermanently)
}

func serve(ch chan error, s Server) {
	ch <- s.Serve()
}
