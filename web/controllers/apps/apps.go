package apps

import (
	"fmt"
	"net/http"

	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/convox/kernel/web/controllers"
	"github.com/convox/kernel/web/helpers"
	"github.com/convox/kernel/web/models"
	"github.com/convox/kernel/web/models/app"
)

func init() {
	controllers.RegisterTemplate("app", "layout", "app")
}

func Show(rw http.ResponseWriter, r *http.Request) {
	cluster := models.Cluster{mux.Vars(r)["cluster"], "running", 20, 100, 60, 100, 5, 100, nil}
	app := models.App{mux.Vars(r)["app"], "unknown", 40, 100, 60, 100, 10, 100, &cluster, make(models.Processes, 2)}
	app.Processes[0] = models.Process{"web", "bundle exec rails start -p $PORT", 3, 60, 100, 20, 100, 5, 100, &app, nil}
	app.Processes[1] = models.Process{"worker", "bin/worker start", 24, 20, 100, 80, 100, 15, 100, &app, nil}
	controllers.RenderTemplate(rw, "app", app)
}

func Create(rw http.ResponseWriter, r *http.Request) {
	form := helpers.ParseForm(r)
	cluster := form["cluster"]
	err := app.Create(cluster, form["name"])
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
