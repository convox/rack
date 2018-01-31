package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/convox/logger"
	"github.com/gorilla/mux"
)

type Context struct {
	context context.Context
	id      string
	logger  *logger.Logger
	request *http.Request
	writer  io.Writer
}

func (c *Context) Context() context.Context {
	return c.context
}

func (c *Context) Error(err error) error {
	c.logger.Error(err)

	w, ok := c.writer.(http.ResponseWriter)
	if !ok {
		fmt.Fprintf(w, "error: %s\n", err)
		return err
	}

	switch t := err.(type) {
	case Error:
		http.Error(w, t.Error(), t.Code)
	case causer:
		http.Error(w, t.Cause().Error(), http.StatusInternalServerError)
	case error:
		http.Error(w, t.Error(), http.StatusForbidden)
	}

	return err
}

func (c *Context) Form(name string) string {
	return c.request.FormValue(name)
}

func (c *Context) Header(name string) string {
	return c.request.Header.Get(name)
}

func (c *Context) Logf(format string, args ...interface{}) {
	c.logger.Logf(format, args...)
}

func (c *Context) Query(name string) string {
	return c.request.URL.Query().Get(name)
}

func (c *Context) RenderJSON(v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	if _, err := c.writer.Write(data); err != nil {
		return err
	}

	return nil
}

func (c *Context) RenderOK() error {
	fmt.Fprintf(c.writer, "ok\n")
	return nil
}

func (c *Context) Request() *http.Request {
	return c.request
}

func (c *Context) Tag(format string, args ...interface{}) {
	c.logger = c.logger.Append(format, args...)
}

func (c *Context) Var(name string) string {
	return mux.Vars(c.request)[name]
}

type causer interface {
	Cause() error
}
