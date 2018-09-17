package stdapi

import (
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/convox/logger"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/gorilla/websocket"
)

var (
	SessionName   = "console"
	SessionSecret = os.Getenv("SESSION_SECRET")
)

type Context struct {
	context  context.Context
	id       string
	logger   *logger.Logger
	name     string
	request  *http.Request
	response http.ResponseWriter
	rvars    map[string]string
	session  sessions.Store
	vars     map[string]interface{}
	ws       *websocket.Conn
}

type Flash struct {
	Kind    string
	Message string
}

func init() {
	gob.Register(Flash{})
}

func NewContext(w http.ResponseWriter, r *http.Request) *Context {
	return &Context{
		context:  r.Context(),
		logger:   logger.New(""),
		request:  r,
		response: w,
		rvars:    map[string]string{},
		session:  sessions.NewCookieStore([]byte(SessionSecret)),
		vars:     map[string]interface{}{},
	}
}

func (c *Context) Body() io.ReadCloser {
	return c.request.Body
}

func (c *Context) BodyJSON(v interface{}) error {
	data, err := ioutil.ReadAll(c.Body())
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, v); err != nil {
		return err
	}

	return nil
}

// func (c *Context) Close() error {
//   return nil
// }

func (c *Context) Context() context.Context {
	return c.context
}

func (c *Context) Flash(kind, message string) error {
	s, err := c.session.Get(c.request, SessionName)
	if err != nil {
		return err
	}

	s.AddFlash(Flash{Kind: kind, Message: message})

	return s.Save(c.request, c.response)
}

func (c *Context) Flashes() ([]Flash, error) {
	s, err := c.session.Get(c.request, SessionName)
	if err != nil {
		return nil, err
	}

	fs := []Flash{}

	for _, f := range s.Flashes() {
		if ff, ok := f.(Flash); ok {
			fs = append(fs, ff)
		}
	}

	if err := s.Save(c.request, c.response); err != nil {
		return nil, err
	}

	return fs, nil
}

func (c *Context) Form(name string) string {
	return c.request.FormValue(name)
}

func (c *Context) Get(name string) interface{} {
	v, ok := c.vars[name]
	if !ok {
		return nil
	}

	return v
}

func (c *Context) Header(name string) string {
	return c.request.Header.Get(name)
}

func (c *Context) Logf(format string, args ...interface{}) {
	c.logger.Logf(format, args...)
}

func (c *Context) Name() string {
	return c.name
}

func (c *Context) Query(name string) string {
	return c.request.URL.Query().Get(name)
}

func (c *Context) Read(data []byte) (int, error) {
	if c.ws == nil {
		return c.Body().Read(data)
	}

	t, r, err := c.ws.NextReader()
	if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
		return 0, io.EOF
	}
	if err != nil {
		return 0, err
	}

	switch t {
	case websocket.TextMessage:
		return r.Read(data)
	case websocket.BinaryMessage:
		return 0, io.EOF
	default:
		return 0, fmt.Errorf("unknown message type: %d\n", t)
	}
}

func (c *Context) Redirect(code int, target string) error {
	http.Redirect(c.response, c.request, target, code)
	return nil
}

func (c *Context) RenderJSON(v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	c.response.Header().Set("Content-Type", "application/json")

	if _, err := c.response.Write(data); err != nil {
		return err
	}

	if _, err := c.response.Write([]byte{10}); err != nil {
		return err
	}

	return nil
}

func (c *Context) RenderOK() error {
	fmt.Fprintf(c.response, "ok\n")
	return nil
}

func (c *Context) RenderTemplate(path string, params interface{}) error {
	return RenderTemplate(c, path, params)
}

func (c *Context) RenderText(t string) error {
	_, err := c.response.Write([]byte(t))
	return err
}

func (c *Context) Request() *http.Request {
	return c.request
}

func (c *Context) Required(names ...string) error {
	missing := []string{}

	for _, n := range names {
		if c.Form(n) == "" {
			missing = append(missing, n)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("parameter required: %s", strings.Join(missing, ", "))
	}

	return nil
}

func (c *Context) Response() http.ResponseWriter {
	return c.response
}

func (c *Context) SessionGet(name string) (string, error) {
	s, err := c.session.Get(c.request, SessionName)
	if err != nil {
		return "", err
	}

	vi, ok := s.Values[name]
	if !ok {
		return "", nil
	}

	vs, ok := vi.(string)
	if !ok {
		return "", fmt.Errorf("session value is not string")
	}

	return vs, nil
}

func (c *Context) SessionSet(name, value string) error {
	s, err := c.session.Get(c.request, SessionName)
	if err != nil {
		return err
	}

	s.Values[name] = value

	return s.Save(c.request, c.response)
}

func (c *Context) Set(name string, value interface{}) {
	c.vars[name] = value
}

func (c *Context) Tag(format string, args ...interface{}) {
	c.logger = c.logger.Append(format, args...)
}

func (c *Context) SetVar(name, value string) {
	c.rvars[name] = value
}

func (c *Context) Value(name string) string {
	if v := c.Form(name); v != "" {
		return v
	}

	if v := c.Header(name); v != "" {
		return v
	}

	return ""
}

func (c *Context) Var(name string) string {
	if v, ok := c.rvars[name]; ok {
		return v
	}
	return mux.Vars(c.request)[name]
}

func (c *Context) Websocket() *websocket.Conn {
	return c.ws
}

func (c *Context) Write(data []byte) (int, error) {
	if c.ws == nil {
		return c.response.Write(data)
	}

	w, err := c.ws.NextWriter(websocket.TextMessage)
	if err != nil {
		return 0, err
	}

	defer w.Close()

	return w.Write(data)
}

type causer interface {
	Cause() error
}
