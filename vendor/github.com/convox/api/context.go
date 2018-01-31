package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/convox/logger"
	"github.com/gorilla/mux"
)

type Context struct {
	r    *http.Request
	w    http.ResponseWriter
	log  *logger.Logger
	tags []string
}

func NewContext(w http.ResponseWriter, r *http.Request, log *logger.Logger) Context {
	return Context{
		r:    r,
		w:    w,
		log:  log,
		tags: []string{},
	}
}

// Fetch the request body as a byte slice.
//   data, err := c.Body()
func (c *Context) Body() ([]byte, *Error) {
	data, err := ioutil.ReadAll(c.r.Body)

	if err != nil {
		return nil, Errorf(403, "could not read request body")
	}

	return data, nil
}

func (c *Context) Form(name string) string {
	c.r.ParseMultipartForm(2048)

	return c.r.FormValue(name)
}

// Output a line to the logs.
//   c.Logf("count=%d success=%t", 10, true)
//   => count=10 success=true
func (c *Context) Logf(format string, a ...interface{}) {
	if len(c.tags) > 0 {
		format = strings.Join(c.tags, " ") + " " + format
	}

	c.log.Logf(format, a...)
}

// Store data to be added to future log lines.
//   c.Tagf("foo=bar test=%d", 5)
//   c.Logf("baz=qux")
//   => foo=bar test=5 baz=qux
func (c *Context) Tagf(format string, a ...interface{}) {
	c.tags = append(c.tags, fmt.Sprintf(format, a...))
}

// Unmarshal request body into an object based on the Content-Type header.
//   err := c.UnmarshalBody(&obj)
func (c *Context) UnmarshalBody(v interface{}) *Error {
	data, err := c.Body()

	if err != nil {
		return err
	}

	switch c.r.Header.Get("Content-Type") {
	case "application/json":
		if err := json.Unmarshal(data, v); err != nil {
			return Errorf(403, "invalid json")
		}

		return nil
	}

	return Errorf(403, "invalid request type")
}

// Get a variable from the request path.
//   id := c.Var("id")
func (c *Context) Var(name string) string {
	return mux.Vars(c.r)[name]
}

func (c *Context) WriteJSON(v interface{}) *Error {
	data, err := json.Marshal(v)

	if err != nil {
		return ServerError(err)
	}

	_, err = c.w.Write(data)

	if err != nil {
		return ServerError(err)
	}

	return nil
}
