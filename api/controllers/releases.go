package controllers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/options"
	"github.com/convox/rack/structs"
	"github.com/gorilla/mux"
)

func ReleaseCreate(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	app := mux.Vars(r)["app"]

	var opts structs.ReleaseCreateOptions

	if err := unmarshalOptions(r, &opts); err != nil {
		return httperr.Server(err)
	}

	rl, err := Provider.ReleaseCreate(app, opts)
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, rl)
}

func ReleaseGet(rw http.ResponseWriter, req *http.Request) *httperr.Error {
	vars := mux.Vars(req)
	app := vars["app"]
	release := vars["release"]

	r, err := Provider.ReleaseGet(app, release)
	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", app)
	}
	if err != nil && strings.HasPrefix(err.Error(), "no such release") {
		return httperr.Errorf(404, err.Error())
	}
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, r)
}

func ReleaseList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	app := mux.Vars(r)["app"]

	opts := structs.ReleaseListOptions{}

	if v := r.URL.Query().Get("limit"); v != "" {
		i, err := strconv.Atoi(v)
		if err != nil {
			return httperr.Errorf(400, "limit must be numeric")
		}
		opts.Count = options.Int(i)
	}

	releases, err := Provider.ReleaseList(app, opts)
	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", app)
	}
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, releases)
}

func ReleasePromote(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)
	app := vars["app"]
	release := vars["release"]

	_, err := Provider.AppGet(app)
	if err != nil {
		if awsError(err) == "ValidationError" {
			return httperr.Errorf(404, "no such app: %s", app)
		}
		return httperr.Server(err)
	}

	if err := Provider.ReleasePromote(app, release); err != nil {
		Provider.EventSend("release:promote", structs.EventSendOptions{Status: "start", Data: map[string]string{"app": app, "id": release}, Error: err.Error()})
		return httperr.Server(err)
	}

	Provider.EventSend("release:promote", structs.EventSendOptions{Status: "start", Data: map[string]string{"app": app, "id": release}})

	rr, err := Provider.ReleaseGet(app, release)
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, rr)
}
