package router

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
)

type TargetRouter interface {
	Certificate(host string) (*tls.Certificate, error)
	RequestBegin(host string) error
	RequestEnd(host string) error
	Route(host string) (string, error)
}

type HTTP struct {
	certs    sync.Map
	listener net.Listener
	port     int
	router   TargetRouter
}

func NewHTTP(port int, router TargetRouter) (*HTTP, error) {
	h := &HTTP{
		certs:  sync.Map{},
		router: router,
		port:   port,
	}

	ln, err := tls.Listen("tcp", fmt.Sprintf(":%d", h.port), &tls.Config{
		GetCertificate: h.generateCertificate,
	})
	if err != nil {
		return nil, err
	}

	h.listener = ln

	return h, nil
}

func (h *HTTP) Close() error {
	if h.listener == nil {
		return nil
	}

	return h.listener.Close()
}

func (h *HTTP) Port() (string, error) {
	_, port, err := net.SplitHostPort(h.listener.Addr().String())
	if err != nil {
		return "", err
	}

	return port, nil
}

func (h *HTTP) ListenAndServe() error {
	s := &http.Server{Handler: h}
	return s.Serve(h.listener)
}

func (h *HTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.router.RequestBegin(r.Host)
	defer h.router.RequestEnd(r.Host)

	target, err := h.router.Route(r.Host)
	if err != nil {
		http.Error(w, err.Error(), 502)
		return
	}

	fmt.Printf("ns=convox.router at=route host=%q method=%q path=%q\n", r.Host, r.Method, r.RequestURI)

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

func (h *HTTP) generateCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	host := hello.ServerName

	v, ok := h.certs.Load(host)
	if ok {
		if c, ok := v.(tls.Certificate); ok {
			return &c, nil
		}
	}

	c, err := h.router.Certificate(host)
	if err != nil {
		return nil, err
	}

	h.certs.Store(host, *c)

	return c, nil
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
