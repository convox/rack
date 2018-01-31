package api

import (
	"bufio"
	"fmt"
	"net"
	"net/http"

	"github.com/convox/logger"
)

type Logger struct {
	*logger.Logger
}

func NewLogger() *Logger {
	return &Logger{
		Logger: logger.New(Namespace),
	}
}

func (l *Logger) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	log := l.Logger.Start()

	ww := &writeWrapper{ResponseWriter: w}

	next(ww, r)

	status := ww.Status()

	if status == 0 {
		status = 200
	}

	log.Logf("method=%q path=%q status=%d bytes=%d", r.Method, r.RequestURI, status, ww.Size())
}

type writeWrapper struct {
	http.ResponseWriter
	size   int
	status int
}

func (ww *writeWrapper) Size() int {
	return ww.size
}

func (ww *writeWrapper) Status() int {
	return ww.status
}

func (ww *writeWrapper) WriteHeader(status int) {
	ww.status = status
	ww.ResponseWriter.WriteHeader(status)
}

func (ww *writeWrapper) Write(data []byte) (int, error) {
	ww.size += len(data)
	return ww.ResponseWriter.Write(data)
}

func (ww *writeWrapper) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := ww.ResponseWriter.(http.Hijacker)

	if !ok {
		return nil, nil, fmt.Errorf("could not hijack connection")
	}

	return hj.Hijack()
}
