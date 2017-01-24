package controllers

import (
	"net/http"

	"github.com/convox/rack/api/httperr"
)

func RackList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	return httperr.Errorf(403, "GET /racks API is not available on a single Rack. Try https://console.convox.com to manage multiple Racks.")
}
