package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"runtime"
	"strings"

	"golang.org/x/net/websocket"

	"github.com/gorilla/mux"
)

type Router struct {
	*mux.Router
	Middleware []Middleware
	Parent     *Router
	Server     *Server
}

func (rt *Router) Route(method, path string, fn HandlerFunc) {
	sig := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
	parts := strings.Split(sig, ".")
	name := parts[len(parts)-1]

	rt.Handle(path, rt.api(name, fn)).Methods(method)
}

func (rt *Router) Use(mw Middleware) {
	rt.Middleware = append(rt.Middleware, mw)
}

func (rt *Router) UseHandlerFunc(fn http.HandlerFunc) {
	rt.Middleware = append(rt.Middleware, func(gn HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request, c *Context) error {
			fn(w, r)
			return gn(w, r, c)
		}
	})
}

func (rt *Router) streamWebsocket(at string, fn StreamFunc) websocket.Handler {
	return func(ws *websocket.Conn) {
		c, err := rt.context(at, ws, ws.Request())
		if err != nil {
			fmt.Printf("err = %+v\n", err)
			return
		}

		if err := fn(ws, c); err != nil {
			fmt.Printf("err = %+v\n", err)
			return
		}
	}
}

func (rt *Router) api(at string, fn HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lw := responseWriter{ResponseWriter: w, code: 200}

		c, err := rt.context(at, &lw, r)
		if err != nil {
			c.Error(err)
			return
		}

		lw.Header().Add("Request-Id", c.id)

		c.logger.Logf("method=%q path=%q agent=%q", r.Method, r.URL.Path, r.UserAgent())

		c.logger = c.logger.Start()

		mw := []Middleware{}

		if rt.Parent != nil {
			mw = append(mw, rt.Parent.Middleware...)
		}

		mw = append(mw, rt.Middleware...)

		fnmw := rt.wrap(fn, mw...)

		if err := fnmw(&lw, r, c); err != nil {
			c.Error(err)
		}

		c.logger.Logf("code=%d bytes=%d", lw.code, lw.bytes)
	}
}

func (rt *Router) context(name string, w io.Writer, r *http.Request) (*Context, error) {
	id, err := Key(12)
	if err != nil {
		return nil, err
	}

	return &Context{
		context: context.WithValue(r.Context(), "request.id", id),
		id:      id,
		logger:  rt.Server.Logger.Prepend("id=%s", id).At(name),
		request: r,
		writer:  w,
	}, nil
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
