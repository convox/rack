package apps

import (
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/convox/kernel/web/controllers"
	"github.com/convox/kernel/web/models/app"
)

func init() {
	controllers.RegisterTemplate("app", "layout", "app")
}

func Show(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	app, err := app.Show(vars["cluster"], vars["app"])
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

	bhost := os.Getenv("BUILDER_PORT_5000_TCP_ADDR")
	bport := os.Getenv("BUILDER_PORT_5000_TCP_PORT")

	res, err := http.PostForm(fmt.Sprintf("http://%s:%s/clusters/%s/apps/%s/build", bhost, bport, cluster, name), url.Values{
		"repo": {repo},
	})

	fmt.Printf("res %+v\n", res)
	fmt.Printf("err %+v\n", err)

	/* TEMP */
	// err = release.Create(cluster, name, "ami-acb1cfc4", map[string]string{})
	// if err != nil {
	//   controllers.RenderError(rw, err)
	//   return
	// }
	// err = release.Deploy(cluster, name, "ami-acb1cfc4")
	// if err != nil {
	//   controllers.RenderError(rw, err)
	//   return
	// }
	/* END TEMP */
	controllers.Redirect(rw, r, fmt.Sprintf("/clusters/%s/apps/%s", cluster, name))
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
