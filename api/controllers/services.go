package controllers

import (
	"net/http"

	"github.com/convox/rack/api/httperr"
	"github.com/gorilla/mux"
)

func ServiceList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	app := mux.Vars(r)["app"]

	ss, err := Provider.ServiceList(app)
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, ss)
}
