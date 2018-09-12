package stdapi

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type HandlerFunc func(c *Context) error

type Middleware func(fn HandlerFunc) HandlerFunc

type Router struct {
	*mux.Router
	Middleware []Middleware
	Parent     *Router
	Server     *Server
}

type Route struct {
	*mux.Route
	Router *Router
}

func (r Route) Subrouter(prefix string, fn func(Router)) {
	fn(Router{
		Parent: r.Router,
		Router: r.PathPrefix(prefix).Subrouter(),
		Server: r.Router.Server,
	})
}

func (rt *Router) MatcherFunc(fn mux.MatcherFunc) Route {
	return Route{
		Route:  rt.Router.MatcherFunc(fn),
		Router: rt,
	}
}

func (rt *Router) Redirect(method, path string, code int, target string) {
	rt.Handle(path, Redirect(code, target)).Methods(method)
}

func (rt *Router) Route(method, path string, fn HandlerFunc) Route {
	switch method {
	case "SOCKET":
		return Route{
			Route:  rt.Handle(path, rt.websocket(fn)).Methods("GET").Headers("Upgrade", "websocket"),
			Router: rt,
		}
	case "ANY":
		return Route{
			Route:  rt.Handle(path, rt.http(fn)),
			Router: rt,
		}
	default:
		return Route{
			Route:  rt.Handle(path, rt.http(fn)).Methods(method),
			Router: rt,
		}
	}
}

func (rt *Router) Static(prefix, path string) Route {
	return Route{
		Route:  rt.PathPrefix(prefix).Handler(http.StripPrefix(prefix, http.FileServer(http.Dir(path)))),
		Router: rt,
	}
}

func (r *Router) Subrouter(prefix string, fn func(Router)) {
	fn(Router{
		Parent: r,
		Router: r.PathPrefix(prefix).Subrouter(),
		Server: r.Server,
	})
}

func (rt *Router) Use(mw Middleware) {
	rt.Middleware = append(rt.Middleware, mw)
}

func (rt *Router) UseHandlerFunc(fn http.HandlerFunc) {
	rt.Middleware = append(rt.Middleware, func(gn HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			fn(c.response, c.request)
			return gn(c)
		}
	})
}

func (rt *Router) context(name string, w http.ResponseWriter, r *http.Request, conn *websocket.Conn) (*Context, error) {
	id, err := generateId(12)
	if err != nil {
		return nil, err
	}

	w.Header().Add("Request-Id", id)

	c := NewContext(w, r)

	c.context = context.WithValue(r.Context(), "request.id", id)
	c.id = id
	c.logger = rt.Server.Logger.Prepend("id=%s", id).At(name)
	c.ws = conn

	return c, nil
}

func (rt *Router) handle(fn HandlerFunc, c *Context) error {
	defer func() {
		if rt.Server.Recover == nil {
			return
		}
		switch t := recover().(type) {
		case error:
			rt.Server.Recover(t)
		case string:
			rt.Server.Recover(fmt.Errorf(t))
		case fmt.Stringer:
			rt.Server.Recover(fmt.Errorf(t.String()))
		case nil:
			return
		default:
			panic(t)
		}
	}()

	c.logger.Logf("method=%q path=%q", c.request.Method, c.request.URL.Path)
	c.logger = c.logger.Start()

	rw := &responseWriter{ResponseWriter: c.response, code: 200}
	c.response = rw

	mw := []Middleware{}

	if rt.Parent != nil {
		mw = append(mw, rt.Parent.Middleware...)
	}

	mw = append(mw, rt.Middleware...)

	fnmw := rt.wrap(fn, mw...)

	err := fnmw(c)

	if ne, ok := err.(*net.OpError); ok {
		c.logger.Logf("state=closed error=%q", ne.Err)
		return nil
	}

	return err
}

func (rt *Router) http(fn HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := rt.context(functionName(fn), w, r, nil)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		if err := rt.handle(fn, c); err != nil {
			switch t := err.(type) {
			case Error:
				c.logger.Append("code=%d", t.Code()).Error(err)
				http.Error(c.response, t.Error(), t.Code())
			case causer:
				c.logger.Error(err)
				http.Error(c.response, "server error", http.StatusInternalServerError)
			case error:
				c.logger.Error(err)
				http.Error(c.response, t.Error(), http.StatusForbidden)
			}
		}
	}
}

var upgrader = websocket.Upgrader{ReadBufferSize: 10 * 1024, WriteBufferSize: 10 * 1024}

func (rt *Router) websocket(fn HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			w.Write([]byte(fmt.Sprintf("ERROR: %s\n", err.Error())))
			return
		}

		// empty binary message signals EOF
		defer conn.WriteMessage(websocket.BinaryMessage, []byte{})
		// defer conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(1000, ""))

		c, err := rt.context(functionName(fn), w, r, conn)
		if err != nil {
			conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("ERROR: %s\n", err.Error())))
			return
		}

		if err := rt.handle(fn, c); err != nil {
			conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("ERROR: %s\n", err.Error())))
			return
		}
	}
}

func (rt *Router) wrap(fn HandlerFunc, m ...Middleware) HandlerFunc {
	if len(m) == 0 {
		return fn
	}

	return m[0](rt.wrap(fn, m[1:len(m)]...))
}

type responseWriter struct {
	http.ResponseWriter
	bytes int64
	code  int
}

func (w *responseWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *responseWriter) Write(data []byte) (int, error) {
	n, err := w.ResponseWriter.Write(data)
	w.bytes += int64(n)
	return n, err
}

func (w *responseWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}
