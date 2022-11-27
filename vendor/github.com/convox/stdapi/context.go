package stdapi

import (
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/convox/logger"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/sebest/xff"
)

var (
	SessionExpiration = 86400 * 30
	SessionName       = ""
	SessionSecret     = ""
)

type Context struct {
	context  context.Context
	id       string
	logger   *logger.Logger
	name     string
	request  *http.Request
	response *Response
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
	s := sessions.NewCookieStore([]byte(SessionSecret))
	s.Options.MaxAge = SessionExpiration
	s.Options.SameSite = http.SameSiteLaxMode

	return &Context{
		context:  r.Context(),
		logger:   logger.New(""),
		request:  r,
		response: &Response{ResponseWriter: w},
		rvars:    map[string]string{},
		session:  s,
		vars:     map[string]interface{}{},
	}
}

func (c *Context) Ajax() bool {
	return c.request.Header.Get("X-Requested-With") == "XMLHttpRequest"
}

func (c *Context) Body() io.ReadCloser {
	return c.request.Body
}

func (c *Context) BodyJSON(v interface{}) error {
	data, err := ioutil.ReadAll(c.Body())
	if err != nil {
		return errors.WithStack(err)
	}

	if err := json.Unmarshal(data, v); err != nil {
		return errors.WithStack(err)
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
		return errors.WithStack(err)
	}

	s.AddFlash(Flash{Kind: kind, Message: message})

	return s.Save(c.request, c.response)
}

func (c *Context) Flashes() ([]Flash, error) {
	s, err := c.session.Get(c.request, SessionName)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	fs := []Flash{}

	for _, f := range s.Flashes() {
		if ff, ok := f.(Flash); ok {
			fs = append(fs, ff)
		}
	}

	if err := s.Save(c.request, c.response); err != nil {
		return nil, errors.WithStack(err)
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

func (c *Context) IP() string {
	return strings.Split(xff.GetRemoteAddr(c.Request()), ":")[0]
}

func (c *Context) Logger() *logger.Logger {
	return c.logger
}

func (c *Context) Logf(format string, args ...interface{}) {
	c.logger.Logf(format, args...)
}

func (c *Context) Name() string {
	return c.name
}

func (c *Context) Protocol() string {
	if h := c.Header("X-Forwarded-Proto"); h != "" {
		return h
	}

	return "https"
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
		return 0, errors.WithStack(err)
	}

	switch t {
	case websocket.TextMessage:
		return r.Read(data)
	case websocket.BinaryMessage:
		return 0, io.EOF
	default:
		return 0, errors.WithStack(fmt.Errorf("unknown message type: %d\n", t))
	}
}

func (c *Context) Redirect(code int, target string) error {
	http.Redirect(c.response, c.request, target, code)
	return nil
}

func (c *Context) RenderJSON(v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return errors.WithStack(err)
	}

	c.response.Header().Set("Content-Type", "application/json")

	if _, err := c.response.Write(data); err != nil {
		return errors.WithStack(err)
	}

	if _, err := c.response.Write([]byte{10}); err != nil {
		return errors.WithStack(err)
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

func (c *Context) RenderTemplatePart(path, part string, params interface{}) error {
	return RenderTemplatePart(c, path, part, params)
}

func (c *Context) RenderText(t string) error {
	_, err := c.response.Write([]byte(t))
	return errors.WithStack(err)
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
		return errors.WithStack(fmt.Errorf("parameter required: %s", strings.Join(missing, ", ")))
	}

	return nil
}

func (c *Context) Response() *Response {
	return c.response
}

func (c *Context) SessionGet(name string) (string, error) {
	if SessionName == "" {
		return "", fmt.Errorf("no session name set")
	}

	if SessionSecret == "" {
		return "", fmt.Errorf("no session secret set")
	}

	s, _ := c.session.Get(c.request, SessionName)

	vi, ok := s.Values[name]
	if !ok {
		return "", nil
	}

	vs, ok := vi.(string)
	if !ok {
		return "", errors.WithStack(fmt.Errorf("session value is not string"))
	}

	return vs, nil
}

func (c *Context) SessionSet(name, value string) error {
	if SessionName == "" {
		return fmt.Errorf("no session name set")
	}

	if SessionSecret == "" {
		return fmt.Errorf("no session secret set")
	}

	s, _ := c.session.Get(c.request, SessionName)

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
		return 0, errors.WithStack(err)
	}

	defer w.Close()

	return w.Write(data)
}
