package clusters

import (
	"net/http"

	"github.com/convox/convox/kernel/controllers"
	"github.com/convox/convox/kernel/helpers"
	"github.com/convox/convox/kernel/models/cluster"

	"github.com/convox/convox/Godeps/_workspace/src/github.com/gorilla/mux"
)

func init() {
	controllers.RegisterTemplate("clusters", "layout", "clusters")
	controllers.RegisterTemplate("cluster", "layout", "cluster")
}

func List(rw http.ResponseWriter, r *http.Request) {
	clusters, err := cluster.List()
	if err != nil {
		controllers.RenderError(rw, err)
		return
	}
	controllers.RenderTemplate(rw, "clusters", clusters)
}

func Show(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cluster, err := cluster.Show(vars["cluster"])
	if err != nil {
		controllers.RenderError(rw, err)
		return
	}
	controllers.RenderTemplate(rw, "cluster", cluster)
}

func Create(rw http.ResponseWriter, r *http.Request) {
	form := helpers.ParseForm(r)
	err := cluster.Create(form["name"])
	if err != nil {
		controllers.RenderError(rw, err)
		return
	}
	controllers.Redirect(rw, r, "/clusters")
}

func Delete(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["cluster"]
	err := cluster.Delete(name)
	if err != nil {
		controllers.RenderError(rw, err)
		return
	}
	controllers.Redirect(rw, r, "/clusters")
}
