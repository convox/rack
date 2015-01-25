package controllers

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"

	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/convox/kernel/web/models"
)

func init() {
	RegisterTemplate("apps", "layout", "apps")
	RegisterTemplate("app", "layout", "app")
}

func AppList(rw http.ResponseWriter, r *http.Request) {
	apps, err := models.ListApps()

	if err != nil {
		RenderError(rw, err)
		return
	}

	sort.Sort(apps)

	RenderTemplate(rw, "apps", apps)
}

func AppShow(rw http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["app"]

	app, err := models.GetApp(name)

	if err != nil {
		RenderError(rw, err)
		return
	}

	RenderTemplate(rw, "app", app)
}

func AppCreate(rw http.ResponseWriter, r *http.Request) {
	form := ParseForm(r)
	name := form["name"]
	repo := form["repo"]

	app := &models.App{
		Name:       name,
		Repository: repo,
	}

	err := app.Create()

	if err != nil {
		RenderError(rw, err)
		return
	}

	Redirect(rw, r, "/apps")
}

func AppDelete(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["app"]

	app, err := models.GetApp(name)

	if err != nil {
		RenderError(rw, err)
		return
	}

	err = app.Delete()

	if err != nil {
		RenderError(rw, err)
		return
	}

	RenderText(rw, "ok")
}

func AppBuild(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	form := ParseForm(r)
	app := vars["app"]
	repo := form["repo"]

	bhost := os.Getenv("BUILDER_PORT_5000_TCP_ADDR")
	bport := os.Getenv("BUILDER_PORT_5000_TCP_PORT")

	_, err := http.PostForm(fmt.Sprintf("http://%s:%s/apps/%s/build", bhost, bport, app), url.Values{"repo": {repo}})

	if err != nil {
		RenderError(rw, err)
		return
	}

	Redirect(rw, r, fmt.Sprintf("/apps/%s", app))
}

func AppPromote(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	form := ParseForm(r)
	app := vars["app"]

	release, err := models.GetRelease(app, form["release"])

	if err != nil {
		RenderError(rw, err)
		return
	}

	err = release.Promote()

	if err != nil {
		RenderError(rw, err)
		return
	}

	Redirect(rw, r, fmt.Sprintf("/apps/%s", app))
}
