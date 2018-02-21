package api

import (
	"io"
	"net/http"

	"golang.org/x/net/websocket"
)

type HandlerFunc func(w http.ResponseWriter, r *http.Request, c *Context) error

type Middleware func(fn HandlerFunc) HandlerFunc

type StreamFunc func(rw io.ReadWriteCloser, c *Context) error

type WebsocketFunc func(cn *websocket.Conn) error

func NewHandlerFunc(fn http.HandlerFunc) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, c *Context) error {
		fn(w, r)
		return nil
	}
}
