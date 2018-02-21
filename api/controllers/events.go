package controllers

import (
	"net/http"

	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/structs"
	"github.com/gorilla/mux"
)

func EventSend(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	action := mux.Vars(r)["action"]

	var opts structs.EventSendOptions

	if err := unmarshalOptions(r, &opts); err != nil {
		return httperr.Server(err)
	}

	if err := Provider.EventSend(action, opts); err != nil {
		return httperr.Server(err)
	}

	return RenderSuccess(rw)
}
