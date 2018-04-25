package router

import (
	"crypto/tls"
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"time"

	"github.com/convox/rack/helpers"
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
