package api

import (
	"fmt"
	"net/http"

	"github.com/codegangsta/negroni"
)

func BasicAuth(username, password string) negroni.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		_, pw, ok := r.BasicAuth()

		switch {
		case password == "":
			next(w, r)
		case ok && pw == password:
			next(w, r)
		default:
			w.Header().Set("WWW-Authenticate", fmt.Sprintf("Basic realm=\"%s\"", Namespace))
			http.Error(w, "invalid auth", 401)
		}
	}
}
