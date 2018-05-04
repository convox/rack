package api

import (
	"context"
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

func (rt *Router) Redirect(method, path string, code int, target string) {
	rt.Handle(path, Redirect(code, target)).Methods(method)
}

func (rt *Router) Route(method, path string, fn HandlerFunc) {
	rt.Handle(path, rt.http(fn)).Methods(method)
	rt.Handle(path, rt.websocket(fn)).Methods(method).Headers("Upgrade", "websocket")
}

func (rt *Router) Static(prefix, path string) {
	rt.PathPrefix(prefix).Handler(http.StripPrefix(prefix, http.FileServer(http.Dir(path))))
}

func (r *Router) Subrouter(prefix string) Router {
	return Router{
		Parent: r,
		Router: r.PathPrefix(prefix).Subrouter(),
		Server: r.Server,
	}
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
	c.logger = rt.Server.Logger.Append("id=%s cn=%s", id, name)
	c.ws = conn

	return c, nil
}

func (rt *Router) handle(fn HandlerFunc, c *Context) error {
	c.logger.At("start").Logf("method=%q path=%q", c.request.Method, c.request.URL.Path)
	c.logger = c.logger.Start()

	rw := &responseWriter{ResponseWriter: c.response, code: 200}
	c.response = rw

	mw := []Middleware{}

	if rt.Parent != nil {
		mw = append(mw, rt.Parent.Middleware...)
	}

	mw = append(mw, rt.Middleware...)

	fnmw := rt.wrap(fn, mw...)

	if err := fnmw(c); err != nil {
		c.Error(err)
	}

	c.logger.At("end").Logf("code=%d", rw.code)

	return nil
}

func (rt *Router) http(fn HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := rt.context(functionName(fn), w, r, nil)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		if err := rt.handle(fn, c); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	}
}

var upgrader = websocket.Upgrader{ReadBufferSize: 1024, WriteBufferSize: 1024}

func (rt *Router) websocket(fn HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		c, err := rt.context(functionName(fn), w, r, conn)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		if err := rt.handle(fn, c); err != nil {
			http.Error(w, err.Error(), 500)
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
