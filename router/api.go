package router

import (
	"strconv"
	"time"

	"github.com/convox/stdapi"
)

func (rt *Router) EndpointCreate(c *stdapi.Context) error {
	r, err := rt.Rack(c.Var("rack"))
	if err != nil {
		return err
	}

	h, err := r.Host(c.Var("host"))
	if err != nil {
		return err
	}

	protocol := c.Form("protocol")
	port := c.Form("port")

	pi, err := strconv.Atoi(port)
	if err != nil {
		return err
	}

	for _, e := range h.Endpoints {
		if e.Port == pi {
			if err := e.Close(); err != nil {
				return err
			}
		}
	}

	e, err := h.NewEndpoint(protocol, pi)
	if err != nil {
		return err
	}

	return c.RenderJSON(e)
}

func (rt *Router) EndpointGet(c *stdapi.Context) error {
	r, err := rt.Rack(c.Var("rack"))
	if err != nil {
		return err
	}

	h, err := r.Host(c.Var("host"))
	if err != nil {
		return err
	}

	pi, err := strconv.Atoi(c.Var("port"))
	if err != nil {
		return err
	}

	e, err := h.Endpoint(pi)
	if err != nil {
		return err
	}

	return c.RenderJSON(e)
}

func (rt *Router) HostCreate(c *stdapi.Context) error {
	r, err := rt.Rack(c.Var("rack"))
	if err != nil {
		return err
	}

	hostname := c.Form("hostname")

	for _, h := range r.Hosts {
		if h.Hostname == hostname {
			return c.RenderJSON(h)
		}
	}

	t, err := r.NewHost(hostname)
	if err != nil {
		return err
	}

	return c.RenderJSON(t)
}

func (rt *Router) HostGet(c *stdapi.Context) error {
	r, err := rt.Rack(c.Var("rack"))
	if err != nil {
		return err
	}

	h, err := r.Host(c.Var("host"))
	if err != nil {
		return err
	}

	return c.RenderJSON(h)
}

func (rt *Router) HostList(c *stdapi.Context) error {
	r, err := rt.Rack(c.Var("rack"))
	if err != nil {
		return err
	}

	return c.RenderJSON(r.Hosts)
}

func (rt *Router) RackCreate(c *stdapi.Context) error {
	endpoint := c.Form("endpoint")
	name := c.Form("name")

	for i, r := range rt.racks {
		if r.Name == name {
			if err := rt.racks[i].Close(); err != nil {
				return err
			}
		}
	}

	r, err := rt.NewRack(endpoint, name)
	if err != nil {
		return err
	}

	return c.RenderJSON(r)
}

func (rt *Router) RackGet(c *stdapi.Context) error {
	r, err := rt.Rack(c.Var("rack"))
	if err != nil {
		return err
	}

	return c.RenderJSON(r)
}

func (rt *Router) RackList(c *stdapi.Context) error {
	return c.RenderJSON(rt.racks)
}

func (rt *Router) TargetAdd(c *stdapi.Context) error {
	r, err := rt.Rack(c.Var("rack"))
	if err != nil {
		return err
	}

	h, err := r.Host(c.Var("host"))
	if err != nil {
		return err
	}

	pi, err := strconv.Atoi(c.Var("port"))
	if err != nil {
		return err
	}

	e, err := h.Endpoint(pi)
	if err != nil {
		return err
	}

	target := c.Form("target")

	if err := e.TargetAdd(target); err != nil {
		return err
	}

	h.Activity = time.Now().UTC()

	return c.RenderOK()
}

func (rt *Router) TargetList(c *stdapi.Context) error {
	r, err := rt.Rack(c.Var("rack"))
	if err != nil {
		return err
	}

	h, err := r.Host(c.Var("host"))
	if err != nil {
		return err
	}

	pi, err := strconv.Atoi(c.Var("port"))
	if err != nil {
		return err
	}

	e, err := h.Endpoint(pi)
	if err != nil {
		return err
	}

	return c.RenderJSON(e.Targets)
}

func (rt *Router) TargetRemove(c *stdapi.Context) error {
	r, err := rt.Rack(c.Var("rack"))
	if err != nil {
		return err
	}

	h, err := r.Host(c.Var("host"))
	if err != nil {
		return err
	}

	pi, err := strconv.Atoi(c.Var("port"))
	if err != nil {
		return err
	}

	e, err := h.Endpoint(pi)
	if err != nil {
		return err
	}

	target := c.Form("target")

	if err := e.TargetDelete(target); err != nil {
		return err
	}

	return c.RenderOK()
}
