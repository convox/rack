package controllers

import (
	"net/http"

	"github.com/convox/rack/api/httperr"
)

func Auth(w http.ResponseWriter, r *http.Request) *httperr.Error {
	w.Write([]byte("OK\n"))
	return nil
}
