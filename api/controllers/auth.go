package controllers

import (
	"net/http"

	"github.com/convox/rack/api/httperr"
)

func Auth(w http.ResponseWriter, r *http.Request) *httperr.Error {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte('{"success":true}'))
	return nil
}
