package controllers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/structs"
	"github.com/gorilla/mux"
)

func ReleaseList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	app := mux.Vars(r)["app"]

	var err error
	var limit = 20
	if l := r.URL.Query().Get("limit"); l != "" {
		limit, err = strconv.Atoi(l)
		if err != nil {
			return httperr.Errorf(400, "limit must be numeric")
		}
	}

	releases, err := Provider.ReleaseList(app, structs.ReleaseListOptions{Count: limit})
	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", app)
	}
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, releases)
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

	event := &structs.Event{
		Action: "release:promote",
		Status: "start",
		Data:   map[string]string{"app": app, "id": release},
	}

	Provider.EventSend(event, nil)

	if err := Provider.ReleasePromote(app, release); err != nil {
		Provider.EventSend(event, err)
		return httperr.Server(err)
	}

	rr, err := Provider.ReleaseGet(app, release)
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, rr)
}
