package router

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/convox/api"
)

const (
	cleanupInterval = 5 * time.Second
	cleanupAge      = 60 * time.Second
)

type Endpoint struct {
	Expires time.Time     `json:"expires"`
	Host    string        `json:"host"`
	IP      net.IP        `json:"ip"`
	Proxies map[int]Proxy `json:"proxies"`

	router *Router
}

type Router struct {
	Domain    string
	Interface string
	Subnet    string
	Version   string

	ca        tls.Certificate
	dns       *DNS
	endpoints map[string]Endpoint
	lock      sync.Mutex
	ip        net.IP
	net       *net.IPNet
}

func New(version, domain, iface, subnet string) (*Router, error) {
	ip, net, err := net.ParseCIDR(subnet)
	if err != nil {
		return nil, err
	}

	r := &Router{
		Domain:    domain,
		Interface: iface,
		Subnet:    subnet,
		Version:   version,
		endpoints: map[string]Endpoint{},
		ip:        ip,
		net:       net,
	}

	ca, err := caCertificate()
	if err != nil {
		return nil, err
	}

	r.ca = ca

	d, err := r.NewDNS()
	if err != nil {
		return nil, err
	}

	r.dns = d

	fmt.Printf("ns=convox.router at=new version=%q domain=%q iface=%q subnet=%q\n", r.Version, r.Domain, r.Interface, r.Subnet)

	go r.cleanupTick()

	return r, nil
}

func (r *Router) Serve() error {
	destroyInterface(r.Interface)

	if err := createInterface(r.Interface, r.ip.String()); err != nil {
		return err
	}

	defer destroyInterface(r.Interface)

	// reserve one ip for router
	r.endpoints[fmt.Sprintf("router.%s", r.Domain)] = Endpoint{IP: r.ip}

	rh := fmt.Sprintf("rack.%s", r.Domain)

	ep, err := r.createEndpoint(rh, true)
	if err != nil {
		return err
	}

	if _, err := r.createProxy(rh, fmt.Sprintf("https://%s:443", ep.IP), "https://localhost:5443"); err != nil {
		return err
	}

	go func() {
		logError(r.dns.Serve())
	}()

	a := api.New("convox.router", fmt.Sprintf("router.%s", r.Domain))

	a.Route("GET", "/endpoints", r.EndpointList)
	a.Route("POST", "/endpoints/{host}", r.EndpointCreate)
	a.Route("DELETE", "/endpoints/{host}", r.EndpointDelete)
	a.Route("POST", "/endpoints/{host}/proxies/{port}", r.ProxyCreate)
	a.Route("POST", "/terminate", r.Terminate)
	a.Route("GET", "/version", r.VersionGet)

	if err := a.Listen("https", fmt.Sprintf("%s:443", r.ip)); err != nil {
		return err
	}

	return nil
}

func (r *Router) cleanupTick() {
	tick := time.Tick(cleanupInterval)

	for range tick {
		for host, ep := range r.endpoints {
			if ep.Expires.IsZero() {
				continue
			}

			if ep.Expires.Before(time.Now()) {
				fmt.Printf("ns=convox.router at=cleanup endpoint=%q\n", host)

				if err := r.destroyEndpoint(host); err != nil {
					logError(err)
				}
			}
		}
	}
}

func (r *Router) createEndpoint(host string, system bool) (*Endpoint, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if ep, ok := r.endpoints[host]; ok {
		ep.Expires = time.Now().Add(cleanupAge).UTC()
		r.endpoints[host] = ep
		return &ep, nil
	}

	ip, err := r.nextIP()
	if err != nil {
		return nil, err
	}

	if err := createAlias(r.Interface, ip.String()); err != nil {
		return nil, err
	}

	e := Endpoint{
		Host:    host,
		IP:      ip,
		Proxies: map[int]Proxy{},
		router:  r,
	}

	if !system {
		e.Expires = time.Now().Add(cleanupAge).UTC()
	}

	r.endpoints[host] = e

	return &e, nil
}

func (r *Router) destroyEndpoint(host string) error {
	if ep, ok := r.endpoints[host]; ok {
		for _, p := range ep.Proxies {
			if err := p.Terminate(); err != nil {
				logError(err)
			}
		}
		delete(r.endpoints, host)
		return nil
	}

	return fmt.Errorf("no such endpoint: %s", host)
}

func (r *Router) matchEndpoint(host string) (*Endpoint, error) {
	if ep, ok := r.endpoints[host]; ok {
		return &ep, nil
	}

	parts := strings.Split(host, ".")

	if len(parts) < 3 {
		return nil, fmt.Errorf("no such endpoint: %s", host)
	}

	base := strings.Join(parts[len(parts)-3:len(parts)], ".")
	ep := r.endpoints[base]

	return &ep, nil
}

func (r *Router) createProxy(host, listen, target string) (*Proxy, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	ep, ok := r.endpoints[host]
	if !ok {
		return nil, fmt.Errorf("no such endpoint: %s", host)
	}

	ul, err := url.Parse(listen)
	if err != nil {
		return nil, err
	}

	ut, err := url.Parse(target)
	if err != nil {
		return nil, err
	}

	pi, err := strconv.Atoi(ul.Port())
	if err != nil {
		return nil, err
	}

	if p, ok := r.endpoints[host].Proxies[pi]; ok {
		return &p, nil
	}

	p, err := ep.NewProxy(host, ul, ut)
	if err != nil {
		return nil, err
	}

	r.endpoints[host].Proxies[pi] = *p

	go p.Serve()

	return p, nil
}

func (r *Router) hasIP(ip net.IP) bool {
	for _, e := range r.endpoints {
		if e.IP.Equal(ip) {
			return true
		}
	}

	return false
}

func (r *Router) nextIP() (net.IP, error) {
	ip := make(net.IP, len(r.ip))
	copy(ip, r.ip)

	for {
		if !r.hasIP(ip) {
			break
		}

		ip = incrementIP(ip)
	}

	return ip, nil
}

func incrementIP(ip net.IP) net.IP {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			break
		}
	}

	return ip
}

func execute(command string, args ...string) error {
	cmd := exec.Command(command, args...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	return ioutil.WriteFile(path, data, 0644)
}
