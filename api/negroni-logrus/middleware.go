package negronilogrus

import (
	"fmt"
	"net/http"
	"time"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/codegangsta/negroni"
)

type timer interface {
	Now() time.Time
	Since(time.Time) time.Duration
}

type realClock struct{}

func (rc *realClock) Now() time.Time {
	return time.Now()
}

func (rc *realClock) Since(t time.Time) time.Duration {
	return time.Since(t)
}

// Middleware is a middleware handler that logs the request as it goes in and the response as it goes out.
type Middleware struct {
	// Logger is the log.Logger instance used to log messages with the Logger middleware
	Logger *logrus.Logger
	// Name is the name of the application as recorded in latency metrics
	Name string

	logStarting bool

	clock timer
}

// NewMiddleware returns a new *Middleware, yay!
func NewMiddleware() *Middleware {
	return NewCustomMiddleware(logrus.InfoLevel, &logrus.TextFormatter{}, "web")
}

// NewCustomMiddleware builds a *Middleware with the given level and formatter
func NewCustomMiddleware(level logrus.Level, formatter logrus.Formatter, name string) *Middleware {
	log := logrus.New()
	log.Level = level
	log.Formatter = formatter

	return &Middleware{Logger: log, Name: name, logStarting: true, clock: &realClock{}}
}

// NewMiddlewareFromLogger returns a new *Middleware which writes to a given logrus logger.
func NewMiddlewareFromLogger(logger *logrus.Logger, name string) *Middleware {
	return &Middleware{Logger: logger, Name: name, logStarting: true, clock: &realClock{}}
}

// SetLogStarting accepts a bool to control the logging of "started handling
// request" prior to passing to the next middleware
func (l *Middleware) SetLogStarting(v bool) {
	l.logStarting = v
}

func (l *Middleware) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	start := l.clock.Now()

	// Try to get the real IP
	remoteAddr := r.RemoteAddr
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		remoteAddr = realIP
	}

	entry := l.Logger.WithFields(logrus.Fields{
		"request": r.RequestURI,
		"method":  r.Method,
		"remote":  remoteAddr,
	})

	if reqID := r.Header.Get("X-Request-Id"); reqID != "" {
		entry = entry.WithField("request_id", reqID)
	}

	if l.logStarting {
		entry.Info("started handling request")
	}

	next(rw, r)

	latency := l.clock.Since(start)
	res := rw.(negroni.ResponseWriter)
	entry.WithFields(logrus.Fields{
		"status":                                          res.Status(),
		"text_status":                                     http.StatusText(res.Status()),
		fmt.Sprintf("measure#%s.elapsed", l.Name):         fmt.Sprintf("%0.3fms", float64(latency.Nanoseconds())/1000000),
		fmt.Sprintf("count#status%dXX", res.Status()/100): 1,
	}).Info("completed handling request")
}
