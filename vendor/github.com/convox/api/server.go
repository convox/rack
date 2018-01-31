package api

import (
	"net/http"

	"github.com/codegangsta/negroni"
)

type Server struct {
	*negroni.Negroni
	log *Logger
}

func NewServer() Server {
	server := Server{
		Negroni: negroni.New(),
		log:     NewLogger(),
	}

	server.Use(server.log)

	return server
}

func (s *Server) Listen(addr string) {
	s.log.Logf("listen=%q", addr)

	if err := http.ListenAndServe(addr, s.Negroni); err != nil {
		s.log.Error(err)
	}
}
