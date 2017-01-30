package controllers

import (
	"net/http"

	"github.com/convox/rack/api/httperr"
)

func Switch(w http.ResponseWriter, r *http.Request) *httperr.Error {
	return httperr.Errorf(403, "Your CLI is pointing directly at a Rack. To log into Console instead, run `convox login`")
}
