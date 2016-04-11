package controllers

import (
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/api/models"
	"github.com/convox/rack/api/provider"
	"github.com/gorilla/mux"
)

func ReleaseList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	app := mux.Vars(r)["app"]

	releases, err := provider.ReleaseList(app)
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

	r, err := provider.ReleaseGet(app, release)
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

	_, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", app)
	}

	rr, err := models.GetRelease(app, release)

	if err != nil && strings.HasPrefix(err.Error(), "no such release") {
		return httperr.Errorf(404, "no such release: %s", release)
	}

	if err != nil {
		return httperr.Server(err)
	}

	err = rr.Promote()

	if awsError(err) == "ValidationError" {
		message := err.(awserr.Error).Message()
		return httperr.Errorf(403, message)
	}

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, rr)
}
