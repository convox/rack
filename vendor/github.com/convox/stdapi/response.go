package stdapi

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
)

type Response struct {
	http.ResponseWriter
	code int
}

func (r *Response) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("hijack not supported for %T", r.ResponseWriter)
	}
	return h.Hijack()
}

func (r *Response) Code() int {
	return r.code
}

func (r *Response) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (r *Response) WriteHeader(code int) {
	r.code = code
	r.ResponseWriter.WriteHeader(code)
}
