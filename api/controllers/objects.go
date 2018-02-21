package controllers

import (
	"io"
	"net/http"

	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/structs"
	"github.com/gorilla/mux"
)

func ObjectFetch(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	v := mux.Vars(r)
	app := v["app"]
	key := v["key"]

	or, err := Provider.ObjectFetch(app, key)
	if err != nil {
		return httperr.Server(err)
	}

	if _, err := io.Copy(rw, or); err != nil {
		return httperr.Server(err)
	}

	return nil
}

func ObjectStore(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	v := mux.Vars(r)
	app := v["app"]
	key := v["key"]

	o, err := Provider.ObjectStore(app, key, r.Body, structs.ObjectStoreOptions{})
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, o)
}
