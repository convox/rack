package router

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/convox/api"
)

func (rt *Router) EndpointCreate(w http.ResponseWriter, r *http.Request, c *api.Context) error {
	host := c.Var("host")

	ep, err := rt.createEndpoint(host, false)
	if err != nil {
		return err
	}

	return c.RenderJSON(ep)
}

func (rt *Router) EndpointDelete(w http.ResponseWriter, r *http.Request, c *api.Context) error {
	host := c.Var("host")

	return rt.destroyEndpoint(host)
}

func (rt *Router) EndpointList(w http.ResponseWriter, r *http.Request, c *api.Context) error {
	return c.RenderJSON(rt.endpoints)
}

func (rt *Router) ProxyCreate(w http.ResponseWriter, r *http.Request, c *api.Context) error {
	host := c.Var("host")
	port := c.Var("port")
	scheme := c.Form("scheme")
	target := c.Form("target")

	ep, ok := rt.endpoints[host]
	if !ok {
		return fmt.Errorf("no such endpoint: %s", host)
	}

	p, err := rt.createProxy(host, fmt.Sprintf("%s://%s:%s", scheme, ep.IP, port), target)
	if err != nil {
		return err
	}

	return c.RenderJSON(p)
}

func (rt *Router) Terminate(w http.ResponseWriter, r *http.Request, c *api.Context) error {
	go func() {
		time.Sleep(1 * time.Second)
		fmt.Printf("ns=convox.router at=terminate exit=0\n")
		os.Exit(0)
	}()
	return c.RenderOK()
}

func (rt *Router) VersionGet(w http.ResponseWriter, r *http.Request, c *api.Context) error {
	return c.RenderJSON(map[string]string{
		"version": rt.Version,
	})
}
