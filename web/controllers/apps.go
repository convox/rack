package controllers

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"time"

	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/gorilla/websocket"

	"github.com/convox/kernel/web/models"
)

func init() {
	RegisterPartial("app", "builds")
	RegisterPartial("app", "events")
	RegisterPartial("app", "logs")
	RegisterPartial("app", "releases")
	RegisterPartial("app", "resources")
	RegisterPartial("app", "services")

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

	bhost := os.Getenv("BUILDER_PORT_3000_TCP_ADDR")
	bport := os.Getenv("BUILDER_PORT_3000_TCP_PORT")

	_, err := http.PostForm(fmt.Sprintf("http://%s:%s/apps/%s/build", bhost, bport, app), url.Values{"repo": {repo}})

	if err != nil {
		RenderError(rw, err)
		return
	}

	RenderText(rw, "ok")
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

	event := &models.Event{
		App:      app,
		Created:  time.Now(),
		Metadata: "{}",
		Type:     "PROMOTE",
		State:    "PENDING",
		User:     "web",
	}

	event.Save()

	err = release.Promote()

	if err != nil {
		event.State = "ERROR"
		event.Metadata = err.Error()
		event.Save()

		RenderError(rw, err)
		return
	}

	Redirect(rw, r, fmt.Sprintf("/apps/%s", app))
}

func AppBuilds(rw http.ResponseWriter, r *http.Request) {
	app := mux.Vars(r)["app"]

	builds, err := models.ListBuilds(app)

	if err != nil {
		RenderError(rw, err)
		return
	}

	RenderPartial(rw, "app", "builds", builds)
}

func AppEvents(rw http.ResponseWriter, r *http.Request) {
	app := mux.Vars(r)["app"]

	events, err := models.ListEvents(app)

	if err != nil {
		RenderError(rw, err)
		return
	}

	RenderPartial(rw, "app", "events", events)
}

func AppLogs(rw http.ResponseWriter, r *http.Request) {
	app := mux.Vars(r)["app"]

	RenderPartial(rw, "app", "logs", app)
}

func AppLogStream(rw http.ResponseWriter, r *http.Request) {
	app, err := models.GetApp(mux.Vars(r)["app"])

	if err != nil {
		RenderError(rw, err)
		return
	}

	logs := make(chan []byte)
	done := make(chan bool)

	app.SubscribeLogs(logs, done)

	ws, err := upgrader.Upgrade(rw, r, nil)

	if err != nil {
		RenderError(rw, err)
		return
	}

	defer ws.Close()

	for data := range logs {
		ws.WriteMessage(websocket.TextMessage, data)
	}

	fmt.Println("ended")
}

func AppReleases(rw http.ResponseWriter, r *http.Request) {
	app := mux.Vars(r)["app"]

	releases, err := models.ListReleases(app)

	if err != nil {
		RenderError(rw, err)
		return
	}

	RenderPartial(rw, "app", "releases", releases)
}

func AppResources(rw http.ResponseWriter, r *http.Request) {
	app := mux.Vars(r)["app"]

	resources, err := models.ListResources(app)

	if err != nil {
		RenderError(rw, err)
		return
	}

	RenderPartial(rw, "app", "resources", resources)
}

func AppServices(rw http.ResponseWriter, r *http.Request) {
	app := mux.Vars(r)["app"]

	services, err := models.ListServices(app)

	if err != nil {
		RenderError(rw, err)
		return
	}

	RenderPartial(rw, "app", "services", services)
}

func AppStatus(rw http.ResponseWriter, r *http.Request) {
	app, err := models.GetApp(mux.Vars(r)["app"])

	if err != nil {
		RenderError(rw, err)
		return
	}

	RenderText(rw, app.Status)
}
