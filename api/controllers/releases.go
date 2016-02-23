package controllers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/api/models"
	"github.com/convox/rack/api/provider"
)

func ReleaseList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)
	app := vars["app"]

	_, err := provider.AppGet(app)

	if err != nil {
		return httperr.Server(err)
	}

	if err != nil {
		return httperr.Server(err)
	}

	releases, err := provider.ReleaseList(app)

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, releases)
}

func ReleaseShow(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)
	app := vars["app"]
	release := vars["release"]

	_, err := provider.AppGet(app)

	if err != nil {
		return httperr.Server(err)
	}

	rr, err := provider.ReleaseGet(app, release)

	if err != nil && strings.HasPrefix(err.Error(), "no such release") {
		return httperr.Errorf(404, "no such release: %s", release)
	}

	fmt.Printf("err %+v\n", err)

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, rr)
}

func ReleasePromote(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)
	app := vars["app"]
	release := vars["release"]

	_, err := provider.AppGet(app)

	if err != nil {
		return httperr.Server(err)
	}

	rr, err := provider.ReleaseGet(app, release)

	if err != nil && strings.HasPrefix(err.Error(), "no such release") {
		return httperr.Errorf(404, "no such release: %s", release)
	}

	if err != nil {
		return httperr.Server(err)
	}

	err = models.ReleasePromote(rr)

	if awsError(err) == "ValidationError" {
		message := err.(awserr.Error).Message()
		return httperr.Errorf(403, message)
	}

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, rr)
}
