package router

import (
	"crypto/tls"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/convox/rack/helpers"
	"golang.org/x/net/websocket"
)

type Proxy struct {
	Hostname string
	Listen   *url.URL

	closed   bool
	listener net.Listener
	rfn      ProxyRequestFunc
	router   *Router
	tfn      TargetFetchFunc
}

type TargetFetchFunc func() (*url.URL, error)
type ProxyRequestFunc func(*Proxy, *url.URL)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func (rt *Router) NewProxy(hostname string, listen *url.URL, tfn TargetFetchFunc, rfn ProxyRequestFunc) (*Proxy, error) {
	p := &Proxy{
		Hostname: hostname,
		Listen:   listen,
		tfn:      tfn,
		rfn:      rfn,
	}

	p.router = rt

	return p, nil
}

func (p *Proxy) Close() error {
	p.closed = true
	return p.listener.Close()
}

func (p *Proxy) Serve() error {
	ln, err := net.Listen("tcp", p.Listen.Host)
	if err != nil {
		return err
	}

	defer ln.Close()

	switch p.Listen.Scheme {
	case "https", "tls":
		cert, err := p.router.generateCertificate(p.Hostname)
		if err != nil {
			return err
		}

		cfg := &tls.Config{
			Certificates: []tls.Certificate{cert},
		}

		ln = tls.NewListener(ln, cfg)
	}

	p.listener = ln

	switch p.Listen.Scheme {
	case "http", "https":
		if err := p.proxyHTTP(ln); err != nil {
			return err
		}
	case "tcp", "tls":
		if err := p.proxyTCP(ln); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown listener scheme: %s", p.Listen.Scheme)
	}

	return nil
}

func (p *Proxy) Terminate() error {
	if err := p.listener.Close(); err != nil {
		return err
	}

	return nil
}

func (p *Proxy) proxyHTTP(ln net.Listener) error {
	s := &http.Server{}
	s.Handler = p
	return s.Serve(ln)
}

var hc = &http.Client{
	CheckRedirect: func(r *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
	Transport: &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 10 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	},
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	target, err := p.tfn()
	if err != nil {
		http.Error(w, err.Error(), 502)
		return
	}

	if strings.ToLower(r.Header.Get("Connection")) == "upgrade" {
		websocket.Handler(p.serveWebsocket(r, target)).ServeHTTP(w, r)
		return
	}

	req, err := http.NewRequest(r.Method, fmt.Sprintf("%s%s", target.String(), r.RequestURI), r.Body)
	if err != nil {
		http.Error(w, err.Error(), 502)
		return
	}

	for k, vv := range r.Header {
		for _, v := range vv {
			req.Header.Add(k, v)
		}
	}

	req.Host = r.Host

	req.Header.Add("X-Forwarded-For", r.RemoteAddr)
	req.Header.Set("X-Forwarded-Port", p.Listen.Port())
	req.Header.Set("X-Forwarded-Proto", p.Listen.Scheme)

	res, err := hc.Do(req)
	if err != nil {
		http.Error(w, err.Error(), 502)
		return
	}

	defer res.Body.Close()

	for k, vv := range res.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}

	w.WriteHeader(res.StatusCode)

	if _, err := io.Copy(w, res.Body); err != nil {
		http.Error(w, err.Error(), 502)
		return
	}

	if p.rfn != nil {
		p.rfn(p, target)
	}
}

func (p *Proxy) serveWebsocket(r *http.Request, target *url.URL) websocket.Handler {
	return func(ws *websocket.Conn) {
		wst, err := url.Parse(target.String())
		if err != nil {
			return
		}

		switch target.Scheme {
		case "https":
			wst.Scheme = "wss"
		default:
			wst.Scheme = "ws"
		}

		wst.Path = r.URL.Path

		cn, err := websocket.DialConfig(&websocket.Config{
			Header:   r.Header,
			Location: wst,
			Origin:   target,
			TlsConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			Version: websocket.ProtocolVersionHybi13,
		})
		if err != nil {
			return
		}

		if p.rfn != nil {
			p.rfn(p, target)
		}

		if err := helpers.Pipe(ws, cn); err != nil {
			return
		}
	}
}

func (p *Proxy) proxyTCP(listener net.Listener) error {
	for {
		cn, err := listener.Accept()
		if err != nil {
			if p.closed {
				return nil
			}
			return err
		}

		go p.proxyTCPConnection(cn)
	}
}

func (p *Proxy) proxyTCPConnection(cn net.Conn) error {
	defer cn.Close()

	target, err := p.tfn()
	if err != nil {
		return err
	}

	oc, err := net.Dial("tcp", target.Host)
	if err != nil {
		return err
	}

	defer oc.Close()

	switch target.Scheme {
	case "tls":
		oc = tls.Client(oc, &tls.Config{
			InsecureSkipVerify: true,
		})
	}

	if p.rfn != nil {
		p.rfn(p, target)
	}

	return helpers.Pipe(cn, oc)
}
