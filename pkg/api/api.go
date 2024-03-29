package api

import (
	"net/http"
	"reflect"

	"github.com/convox/rack/pkg/jwt"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/provider"
	"github.com/convox/stdapi"
)

type Server struct {
	*stdapi.Server
	Password string
	Provider structs.Provider
	JwtMngr  *jwt.JwtManager
}

func New() (*Server, error) {
	p, err := provider.FromEnv()
	if err != nil {
		return nil, err
	}

	return NewWithProvider(p), nil
}

func NewWithProvider(p structs.Provider) *Server {
	if err := p.Initialize(structs.ProviderOptions{}); err != nil {
		panic(err)
	}

	key, err := p.SystemJwtSignKey()
	if err != nil {
		panic(err)
	}

	s := &Server{
		Provider: p,
		Server:   stdapi.New("api", "api"),
		JwtMngr:  jwt.NewJwtManager(key),
	}

	s.Server.Router.Router = s.Server.Router.Router.SkipClean(true)

	// s.Router.HandleFunc("/debug/pprof/", pprof.Index)
	// s.Router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	// s.Router.HandleFunc("/debug/pprof/profile", pprof.Profile)
	// s.Router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	// s.Router.HandleFunc("/debug/pprof/trace", pprof.Trace)

	s.Subrouter("/", func(auth *stdapi.Router) {
		auth.Route("GET", "/auth", func(c *stdapi.Context) error { return c.RenderOK() })

		auth.Use(s.authenticate)

		s.setupRoutes(*auth)
	})

	return s
}

func (s *Server) authenticate(next stdapi.HandlerFunc) stdapi.HandlerFunc {
	return func(c *stdapi.Context) error {
		username, pass, _ := c.Request().BasicAuth()
		if username == "jwt" && s.JwtMngr != nil {
			data, err := s.JwtMngr.Verify(pass)
			if err != nil {
				return stdapi.Errorf(http.StatusUnauthorized, "invalid authentication: %s", err)
			}
			c.Set(structs.ConvoxRoleParam, data.Role)
		} else {
			if s.Password != "" && s.Password != pass {
				return stdapi.Errorf(http.StatusUnauthorized, "invalid authentication")
			}
			SetReadWriteRole(c)
		}

		return next(c)
	}
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
