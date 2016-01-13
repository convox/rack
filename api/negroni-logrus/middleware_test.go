package negronilogrus

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/codegangsta/negroni"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/stretchr/testify/assert"
)

var (
	nowTime  = time.Now()
	nowToday = nowTime.Format("2006-01-02")
)

type testClock struct{}

func (tc *testClock) Now() time.Time {
	return nowTime
}

func (tc *testClock) Since(time.Time) time.Duration {
	return 10 * time.Microsecond
}

func TestNewMiddleware_Logger(t *testing.T) {
	l := logrus.New()
	mw := NewMiddleware()
	assert.NotEqual(t, fmt.Sprintf("%p", mw.Logger), fmt.Sprintf("%p", l))
}

func TestNewMiddleware_Name(t *testing.T) {
	mw := NewMiddleware()
	assert.Equal(t, "web", mw.Name)
}

func TestNewMiddleware_LoggerFormatter(t *testing.T) {
	mw := NewMiddleware()
	assert.Equal(t, &logrus.TextFormatter{}, mw.Logger.Formatter)
}

func TestNewMiddleware_logStarting(t *testing.T) {
	mw := NewMiddleware()
	assert.True(t, mw.logStarting)
}

func TestNewCustomMiddleware_Name(t *testing.T) {
	mw := NewCustomMiddleware(logrus.DebugLevel, &logrus.JSONFormatter{}, "test")
	assert.Equal(t, "test", mw.Name)
}

func TestNewCustomMiddleware_LoggerFormatter(t *testing.T) {
	f := &logrus.JSONFormatter{}
	mw := NewCustomMiddleware(logrus.DebugLevel, f, "test")
	assert.Equal(t, f, mw.Logger.Formatter)
}

func TestNewCustomMiddleware_LoggerLevel(t *testing.T) {
	l := logrus.DebugLevel
	mw := NewCustomMiddleware(l, &logrus.JSONFormatter{}, "test")
	assert.Equal(t, l, mw.Logger.Level)
}

func TestNewCustomMiddleware_logStarting(t *testing.T) {
	mw := NewCustomMiddleware(logrus.DebugLevel, &logrus.JSONFormatter{}, "test")
	assert.True(t, mw.logStarting)
}

func TestNewMiddlewareFromLogger_Logger(t *testing.T) {
	l := logrus.New()
	mw := NewMiddlewareFromLogger(l, "test")
	assert.Exactly(t, l, mw.Logger)
}

func TestNewMiddlewareFromLogger_Name(t *testing.T) {
	mw := NewMiddlewareFromLogger(logrus.New(), "test")
	assert.Equal(t, "test", mw.Name)
}

func TestNewMiddlewareFromLogger_logStarting(t *testing.T) {
	mw := NewMiddlewareFromLogger(logrus.New(), "test")
	assert.True(t, mw.logStarting)
}

func setupServeHTTP(t *testing.T) (*Middleware, negroni.ResponseWriter, *http.Request) {
	req, err := http.NewRequest("GET", "http://example.com/stuff?rly=ya", nil)
	assert.Nil(t, err)

	req.RequestURI = "http://example.com/stuff?rly=ya"
	req.Method = "GET"
	req.Header.Set("X-Request-Id", "22035D08-98EF-413C-BBA0-C4E66A11B28D")
	req.Header.Set("X-Real-IP", "10.10.10.10")

	mw := NewMiddleware()
	mw.Logger.Formatter = &logrus.JSONFormatter{
		TimestampFormat: "2006-01-02",
	}
	mw.Logger.Out = &bytes.Buffer{}
	mw.clock = &testClock{}

	return mw, negroni.NewResponseWriter(httptest.NewRecorder()), req
}

func TestMiddleware_ServeHTTP(t *testing.T) {
	mw, rec, req := setupServeHTTP(t)
	mw.ServeHTTP(rec, req, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(418)
	})
	lines := strings.Split(strings.TrimSpace(mw.Logger.Out.(*bytes.Buffer).String()), "\n")
	assert.Len(t, lines, 2)
	assert.JSONEq(t,
		fmt.Sprintf(`{"level":"info","method":"GET","msg":"started handling request",`+
			`"remote":"10.10.10.10","request":"http://example.com/stuff?rly=ya",`+
			`"request_id":"22035D08-98EF-413C-BBA0-C4E66A11B28D","time":"%s"}`, nowToday),
		lines[0])
	assert.JSONEq(t,
		fmt.Sprintf(`{"level":"info","method":"GET","msg":"completed handling request",`+
			`"remote":"10.10.10.10","request":"http://example.com/stuff?rly=ya",`+
			`"measure#web.elapsed":"0.010ms","count#status4XX":1,"text_status":"I'm a teapot",`+
			`"status":418,"request_id":"22035D08-98EF-413C-BBA0-C4E66A11B28D","time":"%s"}`, nowToday),
		lines[1])
}

func TestMiddleware_ServeHTTP_logStartingFalse(t *testing.T) {
	mw, rec, req := setupServeHTTP(t)
	mw.SetLogStarting(false)
	mw.ServeHTTP(rec, req, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(418)
	})
	lines := strings.Split(strings.TrimSpace(mw.Logger.Out.(*bytes.Buffer).String()), "\n")
	assert.Len(t, lines, 1)
	assert.JSONEq(t,
		fmt.Sprintf(`{"level":"info","method":"GET","msg":"completed handling request",`+
			`"remote":"10.10.10.10","request":"http://example.com/stuff?rly=ya",`+
			`"measure#web.elapsed":"0.010ms","count#status4XX":1,"text_status":"I'm a teapot",`+
			`"status":418,"request_id":"22035D08-98EF-413C-BBA0-C4E66A11B28D","time":"%s"}`, nowToday),
		lines[0])
}

func TestRealClock_Now(t *testing.T) {
	rc := &realClock{}
	tf := "2006-01-02T15:04:05"
	assert.Equal(t, rc.Now().Format(tf), time.Now().Format(tf))
}

func TestRealClock_Since(t *testing.T) {
	rc := &realClock{}
	now := rc.Now()

	time.Sleep(10 * time.Millisecond)
	since := rc.Since(now)

	assert.Regexp(t, "^1[0-5]\\.[0-9]+ms", since.String())
}
