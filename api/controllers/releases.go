package controllers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/convox/rack/api/models"
	"github.com/gorilla/mux"
)

func ReleaseList(rw http.ResponseWriter, r *http.Request) *HttpError {
	vars := mux.Vars(r)
	app := vars["app"]

	_, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		return HttpErrorf(404, "no such app: %s", app)
	}

	if err != nil {
		return ServerError(err)
	}

	releases, err := models.ListReleases(app)

	if err != nil {
		return ServerError(err)
	}

	return RenderJson(rw, releases)
}

func ReleaseShow(rw http.ResponseWriter, r *http.Request) *HttpError {
	vars := mux.Vars(r)
	app := vars["app"]
	release := vars["release"]

	_, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		return HttpErrorf(404, "no such app: %s", app)
	}

	rr, err := models.GetRelease(app, release)

	if err != nil && strings.HasPrefix(err.Error(), "no such release") {
		return HttpErrorf(404, "no such release: %s", release)
	}

	fmt.Printf("err %+v\n", err)

	if err != nil {
		return ServerError(err)
	}

	return RenderJson(rw, rr)
}

func ReleasePromote(rw http.ResponseWriter, r *http.Request) *HttpError {
	vars := mux.Vars(r)
	app := vars["app"]
	release := vars["release"]

	_, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		return HttpErrorf(404, "no such app: %s", app)
	}

	rr, err := models.GetRelease(app, release)

	if err != nil && strings.HasPrefix(err.Error(), "no such release") {
		return HttpErrorf(404, "no such release: %s", release)
	}

	if err != nil {
		return ServerError(err)
	}

	err = rr.Promote()

	if awsError(err) == "ValidationError" {
		return HttpErrorf(403, err.(awserr.Error).Message())
	}

	if err != nil {
		return ServerError(err)
	}

	return RenderJson(rw, rr)
}
