package controllers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/ddollar/logger"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/gorilla/mux"

	"github.com/convox/kernel/models"
)

func ReleaseList(rw http.ResponseWriter, r *http.Request) error {
	vars := mux.Vars(r)
	app := vars["app"]

	_, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		return RenderNotFound(rw, fmt.Sprintf("no such app: %s", app))
	}

	if err != nil {
		return err
	}

	releases, err := models.ListReleases(app)

	if err != nil {
		return err
	}

	return RenderJson(rw, releases)
}

func ReleaseShow(rw http.ResponseWriter, r *http.Request) error {
	vars := mux.Vars(r)
	app := vars["app"]
	release := vars["release"]

	_, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		return RenderNotFound(rw, fmt.Sprintf("no such app: %s", app))
	}

	rr, err := models.GetRelease(app, release)

	if err != nil && strings.HasPrefix(err.Error(), "no such release") {
		return RenderNotFound(rw, fmt.Sprintf("no such release: %s", release))
	}

	fmt.Printf("err %+v\n", err)

	if err != nil {
		return err
	}

	return RenderJson(rw, rr)
}

func ReleasePromote(rw http.ResponseWriter, r *http.Request) error {
	vars := mux.Vars(r)
	app := vars["app"]
	release := vars["release"]

	_, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		return RenderNotFound(rw, fmt.Sprintf("no such app: %s", app))
	}

	rr, err := models.GetRelease(app, release)

	if err != nil && strings.HasPrefix(err.Error(), "no such release") {
		return RenderNotFound(rw, fmt.Sprintf("no such release: %s", release))
	}

	if err != nil {
		return err
	}

	err = rr.Promote()

	if err != nil {
		return err
	}

	return RenderJson(rw, rr)
}

func releasesLogger(at string) *logger.Logger {
	return logger.New("ns=kernel cn=releases").At(at)
}
