package router

import (
	"fmt"
	"net"
	"net/url"
	"time"
)

type Host struct {
	Activity time.Time `json:"activity"`
	Hostname string    `json:"hostname"`
	IP       net.IP    `json:"ip"`

	Endpoints Endpoints `json:"endpoints"`

	rack *Rack
}

type Hosts []*Host

func (r *Rack) NewHost(hostname string) (*Host, error) {
	ip, err := r.NextIP()
	if err != nil {
		return nil, err
	}

	h := &Host{
		Activity:  time.Now().UTC(),
		Hostname:  hostname,
		IP:        ip,
		Endpoints: Endpoints{},
		rack:      r,
	}

	if err := createAlias(r.router.iface, fmt.Sprintf("%s", h.IP)); err != nil {
		return nil, err
	}

	r.Hosts = append(r.Hosts, h)

	return h, nil
}

func (h *Host) Close() error {
	for _, e := range h.Endpoints {
		if err := e.Close(); err != nil {
			return err
		}
	}

	hosts := Hosts{}

	for i := range h.rack.Hosts {
		if h.rack.Hosts[i].Hostname != h.Hostname {
			hosts = append(hosts, h.rack.Hosts[i])
		}
	}

	h.rack.Hosts = hosts

	return nil
}

func (h *Host) Endpoint(port int) (*Endpoint, error) {
	for _, e := range h.Endpoints {
		if e.Port == port {
			return e, nil
		}
	}

	return nil, fmt.Errorf("no such endpoint: %d", port)
}

func (h *Host) proxyRequest(p *Proxy, target *url.URL) {
	fmt.Printf("ns=convox.router at=proxy listen=%q target=%q\n", p.Listen, target)
	h.Activity = time.Now().UTC()
}
