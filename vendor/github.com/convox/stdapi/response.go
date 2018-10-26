package stdapi

import "net/http"

type Response struct {
	http.ResponseWriter
	code int
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
