package router

import (
	"fmt"
	"net"
)

type Rack struct {
	Name  string `json:"name"`
	IP    net.IP `json:"ip"`
	Hosts Hosts  `json:"hosts"`

	router *Router
}

type Racks []*Rack

func (rt *Router) NewRack(endpoint, name string) (*Rack, error) {
	r := &Rack{
		Name:   name,
		Hosts:  Hosts{},
		IP:     incrementIP(rt.base, uint32((len(rt.racks)+1)*256)),
		router: rt,
	}

	if err := rt.dns.registerDomain(name); err != nil {
		return nil, err
	}

	h, err := r.NewHost("rack")
	if err != nil {
		return nil, err
	}

	e, err := h.NewEndpoint("tls", 443)
	if err != nil {
		return nil, err
	}

	if err := e.TargetAdd(endpoint); err != nil {
		return nil, err
	}

	rt.racks = append(rt.racks, r)

	return r, nil
}

func (r *Rack) Close() error {
	if err := r.router.dns.unregisterDomain(r.Name); err != nil {
		return err
	}

	for _, h := range r.Hosts {
		if err := h.Close(); err != nil {
			return err
		}
	}

	racks := Racks{}

	for i := range r.router.racks {
		if r.router.racks[i].Name != r.Name {
			racks = append(racks, r.router.racks[i])
		}
	}

	r.router.racks = racks

	return nil
}

func (r *Rack) Host(hostname string) (*Host, error) {
	for i := range r.Hosts {
		if r.Hosts[i].Hostname == hostname {
			return r.Hosts[i], nil
		}
	}

	return nil, fmt.Errorf("no such host: %s", hostname)
}
