package router

import (
	"fmt"
	"math/rand"
	"net/url"
	"strings"

	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/sdk"
	"github.com/convox/rack/structs"
)

type Endpoint struct {
	Protocol string   `json:"protocol"`
	Port     int      `json:"port"`
	Targets  []string `json:"targets"`

	host  *Host
	proxy *Proxy
}

type Endpoints []*Endpoint

func (h *Host) NewEndpoint(protocol string, port int) (*Endpoint, error) {
	e := &Endpoint{
		Protocol: protocol,
		Port:     port,
		Targets:  []string{},
		host:     h,
	}

	listen := fmt.Sprintf("%s://%s:%d", protocol, h.IP, port)

	lu, err := url.Parse(listen)
	if err != nil {
		return nil, err
	}

	p, err := h.rack.router.NewProxy(fmt.Sprintf("%s.%s", h.Hostname, h.rack.Name), lu, e.randomTarget, h.proxyRequest)
	if err != nil {
		return nil, err
	}

	e.proxy = p

	go func(p *Proxy) {
		if err := p.Serve(); err != nil {
			fmt.Printf("ns=convox.rack error=%q\n", err)
		}
	}(p)

	h.Endpoints = append(h.Endpoints, e)

	return e, nil
}

func (e *Endpoint) TargetAdd(target string) error {
	e.Targets = append(e.Targets, target)
	return nil
}

func (e *Endpoint) TargetDelete(target string) error {
	for i := range e.Targets {
		if e.Targets[i] == target {
			e.Targets = append(e.Targets[0:i], e.Targets[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("no such target: %s", target)
}

func (e *Endpoint) Close() error {
	endpoints := Endpoints{}

	for i := range e.host.Endpoints {
		if e.host.Endpoints[i].Port != e.Port {
			endpoints = append(endpoints, e.host.Endpoints[i])
		}
	}

	e.host.Endpoints = endpoints

	return e.proxy.Close()
}

func (e *Endpoint) randomTarget() (*url.URL, error) {
	parts := strings.Split(e.host.Hostname, ".")

	if len(parts) > 1 {
		app := parts[len(parts)-1]

		rh, err := e.host.rack.Host("rack")
		if err != nil {
			return nil, err
		}

		rack, err := sdk.New(fmt.Sprintf("https://%s", rh.IP.String()))
		if err != nil {
			return nil, err
		}

		a, err := rack.AppGet(app)
		if err != nil {
			return nil, err
		}

		if a.Sleep == true {
			if err := rack.AppUpdate(app, structs.AppUpdateOptions{Sleep: options.Bool(false)}); err != nil {
				return nil, err
			}
		}
	}

	if len(e.Targets) == 0 {
		return nil, fmt.Errorf("no targets for endpoint: %s:%d", e.host.Hostname, e.Port)
	}

	t := e.Targets[rand.Intn(len(e.Targets))]

	return url.Parse(t)
}
