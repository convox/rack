package stdapi

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"

	"github.com/convox/logger"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

type RecoverFunc func(error, *Context)

type Server struct {
	Check    HandlerFunc
	Hostname string
	Logger   *logger.Logger
	Recover  RecoverFunc
	Router   *Router
	Wrapper  func(h http.Handler) http.Handler

	middleware []Middleware
	server     http.Server
}

func (s *Server) HandleNotFound(fn HandlerFunc) {
	s.Router.HandleNotFound(fn)
}

func (s *Server) Listen(proto, addr string) error {
	s.Logger.At("listen").Logf("hostname=%q proto=%q addr=%q", s.Hostname, proto, addr)

	l, err := net.Listen("tcp", addr)
	if err != nil {
		return errors.WithStack(err)
	}

	switch proto {
	case "h2", "https", "tls":
		config := &tls.Config{
			MinVersion: tls.VersionTLS12,
		}

		if proto == "h2" {
			config.NextProtos = []string{"h2"}
		}

		cert, err := generateSelfSignedCertificate(s.Hostname)
		if err != nil {
			return errors.WithStack(err)
		}

		config.Certificates = append(config.Certificates, cert)

		l = tls.NewListener(l, config)
	}

	var h http.Handler

	if s.Wrapper != nil {
		h = s.Wrapper(s)
	} else {
		h = s
	}

	s.server = http.Server{Handler: h}

	return s.server.Serve(l)
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *Server) MatcherFunc(fn mux.MatcherFunc) *Router {
	return s.Router.MatcherFunc(fn)
}

func (s *Server) Route(method, path string, fn HandlerFunc) Route {
	return s.Router.Route(method, path, fn)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.Router.ServeHTTP(w, r)
}

func (s *Server) Subrouter(prefix string, fn func(*Router)) *Router {
	r := &Router{
		Parent: s.Router,
		Router: s.Router.PathPrefix(prefix).Subrouter(),
		Server: s,
	}

	fn(r)

	return r
}

func (s *Server) Use(mw Middleware) {
	s.Router.Use(mw)
}

func (s *Server) UseHandlerFunc(fn http.HandlerFunc) {
	s.Router.UseHandlerFunc(fn)
}
