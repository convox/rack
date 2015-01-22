package apps

import (
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/convox/kernel/web/controllers"
	"github.com/convox/kernel/web/models/app"
	"github.com/convox/kernel/web/models/release"
)

func init() {
	controllers.RegisterTemplate("app", "layout", "app")
}

func Show(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	app, err := app.Get(vars["cluster"], vars["app"])
	if err != nil {
		controllers.RenderError(rw, err)
		return
	}
	controllers.RenderTemplate(rw, "app", app)
}

func Create(rw http.ResponseWriter, r *http.Request) {
	form := controllers.ParseForm(r)
	name := form["name"]
	cluster := form["cluster"]
	repo := form["repo"]

	options := map[string]string{
		"repo": repo,
	}

	err := app.Create(cluster, name, options)

	if err != nil {
		controllers.RenderError(rw, err)
		return
	}

	controllers.Redirect(rw, r, fmt.Sprintf("/clusters/%s", cluster))
}

func Delete(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cluster := vars["cluster"]
	err := app.Delete(cluster, vars["app"])
	if err != nil {
		controllers.RenderError(rw, err)
		return
	}
	controllers.Redirect(rw, r, fmt.Sprintf("/clusters/%s", cluster))
}

func Build(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	form := controllers.ParseForm(r)
	cluster := vars["cluster"]
	app := vars["app"]
	repo := form["repo"]

	bhost := os.Getenv("BUILDER_PORT_5000_TCP_ADDR")
	bport := os.Getenv("BUILDER_PORT_5000_TCP_PORT")

	_, err := http.PostForm(fmt.Sprintf("http://%s:%s/clusters/%s/apps/%s/build", bhost, bport, cluster, app), url.Values{"repo": {repo}})

	if err != nil {
		controllers.RenderError(rw, err)
		return
	}

	controllers.Redirect(rw, r, fmt.Sprintf("/clusters/%s/apps/%s", cluster, app))
}

func Promote(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	form := controllers.ParseForm(r)
	cluster := vars["cluster"]
	app := vars["app"]

	// id, err := release.Copy(cluster, app, form["release"])

	// if err != nil {
	//   controllers.RenderError(rw, err)
	//   return
	// }

	// err = release.Promote(cluster, app, id)

	err := release.Promote(cluster, app, form["release"])

	if err != nil {
		fmt.Printf("err %+v\n", err)
	}

	controllers.Redirect(rw, r, fmt.Sprintf("/clusters/%s/apps/%s", cluster, app))
}
