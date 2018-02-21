package api

import (
	"fmt"
	"net/http"

	"github.com/convox/logger"
	"github.com/gorilla/mux"
)

type Error struct {
	error
	Code int
}

func New(ns, hostname string) *Server {
	logger := logger.New(fmt.Sprintf("ns=%s", ns))

	server := &Server{
		Hostname: hostname,
		Logger:   logger,
	}

	server.Router = &Router{
		Parent: nil,
		Router: mux.NewRouter(),
		Server: server,
	}

	server.Router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		id, _ := Key(12)
		logger.Logf("id=%s route=unknown code=404 method=%q path=%q", id, r.Method, r.URL.Path)
	})

	return server
}

func Errorf(code int, format string, args ...interface{}) Error {
	return Error{
		error: fmt.Errorf(format, args...),
		Code:  code,
	}
}
