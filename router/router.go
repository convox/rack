package router

import (
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/convox/stdapi"
)

type Router struct {
	base    net.IP
	ca      tls.Certificate
	dns     *DNS
	iface   string
	subnet  string
	net     *net.IPNet
	racks   Racks
	version string
}

func New(iface, subnet, version string) (*Router, error) {
	ip, net, err := net.ParseCIDR(subnet)
	if err != nil {
		return nil, err
	}

	r := &Router{
		base:    ip.To4(),
		iface:   iface,
		net:     net,
		racks:   Racks{},
		subnet:  subnet,
		version: version,
	}

	ca, err := caCertificate()
	if err != nil {
		return nil, err
	}

	r.ca = ca

	return r, nil
}

func (rt *Router) Lookup(host string) net.IP {
	parts := strings.Split(host, ".")

	if len(parts) < 2 {
		return nil
	}

	r, err := rt.Rack(parts[len(parts)-1])
	if err != nil {
		return nil
	}

	h, err := r.Host(strings.Join(parts[0:len(parts)-1], "."))
	if err != nil {
		return nil
	}

	return h.IP
}

var ipLock sync.Mutex

func (rt *Router) NextIP() (net.IP, error) {
	ipLock.Lock()
	defer ipLock.Unlock()

	for i := uint32(1); i < 255; i++ {
		ip := incrementIP(rt.base, (i * 256))
		found := false
		for _, r := range rt.racks {
			if r.IP.Equal(ip) {
				found = true
				break
			}
		}
		if !found {
			return ip, nil
		}
	}
	return net.IP{}, fmt.Errorf("ip exhaustion")
}

func (rt *Router) Rack(name string) (*Rack, error) {
	for i := range rt.racks {
		if rt.racks[i].Name == name {
			return rt.racks[i], nil
		}
	}

	return nil, fmt.Errorf("no such rack: %s", name)
}

func (rt *Router) Serve() error {
	destroyInterface(rt.iface)

	if err := createInterface(rt.iface, rt.base.String()); err != nil {
		return err
	}

	defer destroyInterface(rt.iface)

	if err := createAlias(rt.iface, rt.base.String()); err != nil {
		return err
	}

	d, err := NewDNS(rt.base, rt.Lookup)
	if err != nil {
		return err
	}

	rt.dns = d

	go rt.dns.Serve()

	a := stdapi.New("convox.router", "router")

	a.Route("GET", "/", rt.RackList)
	a.Route("POST", "/racks", rt.RackCreate)
	a.Route("GET", "/racks/{rack}", rt.RackGet)
	a.Route("GET", "/racks", rt.RackList)
	a.Route("POST", "/racks/{rack}/hosts", rt.HostCreate)
	a.Route("GET", "/racks/{rack}/hosts/{host}", rt.HostGet)
	a.Route("GET", "/racks/{rack}/hosts", rt.HostList)
	a.Route("POST", "/racks/{rack}/hosts/{host}/endpoints", rt.EndpointCreate)
	a.Route("GET", "/racks/{rack}/hosts/{host}/endpoints/{port}", rt.EndpointGet)
	a.Route("POST", "/racks/{rack}/hosts/{host}/endpoints/{port}/targets/add", rt.TargetAdd)
	a.Route("GET", "/racks/{rack}/hosts/{host}/endpoints/{port}/targets", rt.TargetList)
	a.Route("POST", "/racks/{rack}/hosts/{host}/endpoints/{port}/targets/delete", rt.TargetRemove)
	a.Route("POST", "/terminate", rt.Terminate)
	a.Route("GET", "/version", rt.Version)

	// a.Route("GET", "/endpoints", r.HostList)
	// a.Route("POST", "/endpoints/{host}", r.HostCreate)
	// a.Route("DELETE", "/endpoints/{host}", r.HostDelete)
	// a.Route("POST", "/endpoints/{host}/proxies/{port}", r.ProxyCreate)
	// a.Route("POST", "/terminate", r.Terminate)
	// a.Route("GET", "/version", r.VersionGet)

	if err := a.Listen("https", fmt.Sprintf("%s:443", rt.base)); err != nil {
		return err
	}

	return nil
}

func (rt *Router) Terminate(c *stdapi.Context) error {
	go func() {
		time.Sleep(1 * time.Second)
		os.Exit(0)
	}()

	return nil
}

func (rt *Router) Version(c *stdapi.Context) error {
	v := map[string]string{
		"version": rt.version,
	}

	return c.RenderJSON(v)
}
