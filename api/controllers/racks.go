package controllers

import (
	"net/http"

	"github.com/convox/rack/api/httperr"
)

func RackList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	return httperr.Errorf(403, "only available on console")
}
