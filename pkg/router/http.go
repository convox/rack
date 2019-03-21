package router

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type HTTP struct {
	listener net.Listener
	router   HTTPRouter
	server   http.Server
}

type HTTPRouter interface {
	RequestBegin(target string) error
	RequestEnd(target string) error
	Route(host string) (string, error)
}

func NewHTTP(ln net.Listener, router HTTPRouter) (*HTTP, error) {
	h := &HTTP{
		router: router,
	}

	h.listener = ln

	fmt.Printf("ns=http at=new\n")

	return h, nil
}

func (h *HTTP) Port() (string, error) {
	_, port, err := net.SplitHostPort(h.listener.Addr().String())
	if err != nil {
		return "", err
	}

	return port, nil
}

func (h *HTTP) Shutdown(ctx context.Context) error {
	fmt.Printf("ns=http at=shutdown\n")

	return h.server.Shutdown(ctx)
}

func (h *HTTP) ListenAndServe() error {
	fmt.Printf("ns=http at=serve\n")

	h.server = http.Server{Handler: h}

	return h.server.Serve(h.listener)
}

func (h *HTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/convox/health" {
		fmt.Fprintf(w, "ok")
		return
	}

	target, err := h.router.Route(r.Host)
	if err != nil {
		http.Error(w, err.Error(), 502)
		return
	}

	h.router.RequestBegin(target)
	defer h.router.RequestEnd(target)

	fmt.Printf("ns=http at=route host=%q method=%q path=%q\n", r.Host, r.Method, r.RequestURI)

	tu, err := url.Parse(target)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid target: %s", target), 502)
		return
	}

	p := httputil.NewSingleHostReverseProxy(tu)

	p.Director = h.proxyDirector(p.Director)

	p.ErrorHandler = h.proxyErrorHandler

	p.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	p.ServeHTTP(w, r)
}

func (h *HTTP) proxyDirector(existing func(r *http.Request)) func(r *http.Request) {
	return func(r *http.Request) {
		existing(r)

		port, err := h.Port()
		if err != nil {
			return
		}

		if v := r.Header.Get("X-Forwarded-Port"); v != "" {
			r.Header.Set("X-Forwarded-Port", v)
		} else {
			r.Header.Set("X-Forwarded-Port", port)
		}

		if v := r.Header.Get("X-Forwarded-Proto"); v != "" {
			r.Header.Set("X-Forwarded-Proto", v)
		} else {
			r.Header.Set("X-Forwarded-Proto", "https")
		}
	}
}

func (h *HTTP) proxyErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	http.Error(w, err.Error(), 502)
}
