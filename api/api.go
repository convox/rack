package api

import (
	"reflect"

	"github.com/convox/rack/provider"
	"github.com/convox/rack/structs"
	"github.com/convox/stdapi"
)

type Server struct {
	*stdapi.Server
	Password string
	Provider structs.Provider
}

func New() *Server {
	return NewWithProvider(provider.FromEnv())
}

func NewWithProvider(p structs.Provider) *Server {
	if err := p.Initialize(structs.ProviderOptions{}); err != nil {
		panic(err)
	}

	s := &Server{
		Provider: p,
		Server:   stdapi.New("api", "api"),
	}

	s.Route("GET", "/check", s.check)

	auth := s.Subrouter("/")

	auth.Use(s.authenticate)

	s.setupRoutes(auth)

	return s
}

func (s *Server) authenticate(next stdapi.HandlerFunc) stdapi.HandlerFunc {
	return func(c *stdapi.Context) error {
		if _, pass, _ := c.Request().BasicAuth(); s.Password != "" && s.Password != pass {
			return stdapi.Errorf(401, "invalid authentication")
		}
		return next(c)
	}
}

func (s *Server) check(c *stdapi.Context) error {
	return c.RenderOK()
}

func (s *Server) hook(name string, args ...interface{}) error {
	vfn, ok := reflect.TypeOf(s).MethodByName(name)
	if !ok {
		return nil
	}

	rargs := []reflect.Value{reflect.ValueOf(s)}

	for _, arg := range args {
		rargs = append(rargs, reflect.ValueOf(arg))
	}

	rvs := vfn.Func.Call(rargs)
	if len(rvs) == 0 {
		return nil
	}

	if err, ok := rvs[0].Interface().(error); ok && err != nil {
		return err
	}

	return nil
}

func (s *Server) provider(c *stdapi.Context) structs.Provider {
	return s.Provider
}
