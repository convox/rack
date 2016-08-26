package controllers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/api/models"
	"github.com/convox/rack/api/structs"
	"github.com/gorilla/mux"
)

func ReleaseList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	app := mux.Vars(r)["app"]

	releases, err := models.Provider().ReleaseList(app, 20)
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

	r, err := models.Provider().ReleaseGet(app, release)
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

// ForkRlease creates a new release based on the app's release
func ForkRelease(app *structs.App) (*structs.Release, error) {
	release := structs.NewRelease(app.Name)

	if app.Release != "" {
		r, err := models.Provider().ReleaseGet(app.Name, app.Release)
		if err != nil {
			return nil, err
		}
		id := release.Id
		created := release.Created

		release = r
		release.Id = id
		release.Created = created
	}

	env, err := models.Provider().EnvironmentGet(app.Name)
	if err != nil {
		fmt.Printf("fn=ForkRelease level=error msg=\"error getting environment: %s\"", err)
	}

	release.Env = env.Raw()

	return &structs.Release{
		Id:       release.Id,
		App:      release.App,
		Build:    release.Build,
		Env:      release.Env,
		Manifest: release.Manifest,
		Created:  release.Created,
	}, nil
}
