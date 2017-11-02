package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/api/models"
	"github.com/convox/rack/api/structs"
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

	releases, err := models.Provider().ReleaseList(app, int64(limit))
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

	event := &structs.Event{
		Action: "release:promote",
		Status: "start",
		Data: map[string]string{
			"app": app,
			"id":  release,
		},
	}

	models.Provider().EventSend(event, nil)

	a, err := models.GetApp(app)
	if err != nil {
		if awsError(err) == "ValidationError" {
			e := httperr.Errorf(404, "no such app: %s", app)
			models.Provider().EventSend(event, e)
			return e
		}

		models.Provider().EventSend(event, err)
		return httperr.Server(err)
	}

	switch a.Tags["Generation"] {
	case "2":
		return releasePromoteGeneration2(rw, r)
	default:
		return releasePromoteGeneration1(rw, r)
	}

	return httperr.Server(fmt.Errorf("unknown generation"))
}

func releasePromoteGeneration1(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)
	app := vars["app"]
	release := vars["release"]

	event := &structs.Event{
		Action: "release:promote",
		Status: "start",
		Data: map[string]string{
			"app": app,
			"id":  release,
		},
	}

	rr, err := models.GetRelease(app, release)
	if err != nil {
		if strings.HasPrefix(err.Error(), "no such release") {
			e := httperr.Errorf(404, "no such release: %s", release)
			models.Provider().EventSend(event, e)
			return e
		}

		models.Provider().EventSend(event, err)
		return httperr.Server(err)
	}

	if err := rr.Promote(); err != nil {
		if awsError(err) == "ValidationError" {
			e := httperr.Errorf(403, err.(awserr.Error).Message())
			models.Provider().EventSend(event, e)
			return e
		}

		models.Provider().EventSend(event, err)
		return httperr.Server(err)
	}

	return RenderJson(rw, rr)
}

func releasePromoteGeneration2(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)
	app := vars["app"]
	release := vars["release"]

	event := &structs.Event{
		Action: "release:promote",
		Status: "start",
		Data: map[string]string{
			"app": app,
			"id":  release,
		},
	}

	rr, err := models.Provider().ReleaseGet(app, release)
	if err != nil {
		models.Provider().EventSend(event, err)
		return httperr.Server(err)
	}

	if err := models.Provider().ReleasePromote(rr); err != nil {
		models.Provider().EventSend(event, err)
		return httperr.Server(err)
	}

	return RenderJson(rw, rr)
}

// ForkRelease creates a new release based on the app's release
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
