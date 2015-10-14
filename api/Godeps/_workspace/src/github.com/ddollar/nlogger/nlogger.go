package nlogger

import (
	"bufio"
	"fmt"
	"net"
	"net/http"

	"github.com/convox/rack/api/Godeps/_workspace/src/github.com/ddollar/logger"
)

type logResponseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *logResponseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *logResponseWriter) Status() int {
	return rw.status
}

func (rw *logResponseWriter) Hijack() (rwc net.Conn, buf *bufio.ReadWriter, err error) {
	hj, ok := rw.ResponseWriter.(http.Hijacker)

	if !ok {
		return nil, nil, fmt.Errorf("could not hijack connection")
	}

	return hj.Hijack()
}

func Log(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
}

func logRequest(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
}

type Nlogger struct {
	ignore []string
	log    *logger.Logger
}

func New(ns string, ignore []string) *Nlogger {
	return &Nlogger{ignore: ignore, log: logger.New(ns)}
}

func (nl *Nlogger) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	rlog := nl.log.Start()

	lrw := &logResponseWriter{ResponseWriter: rw}

	next(lrw, r)

	if nl.ignore != nil {
		for _, path := range nl.ignore {
			if r.RequestURI == path {
				return
			}
		}
	}

	status := lrw.Status()

	if status == 0 {
		status = 200
	}

	rlog.At("request").Success("status=%d method=%q path=%q", status, r.Method, r.RequestURI)
}
