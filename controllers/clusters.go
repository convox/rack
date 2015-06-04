package controllers

import (
	"net/http"
	"os"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/convox/kernel/helpers"
	"github.com/convox/kernel/models"
)

func init() {
	RegisterTemplate("clusters", "layout", "clusters")
	RegisterTemplate("cluster", "layout", "cluster")
}

func ClusterList(rw http.ResponseWriter, r *http.Request) {
	clusters, err := models.ListClusters()

	if err != nil {
		helpers.Error(nil, err)
		RenderError(rw, err)
		return
	}

	RenderTemplate(rw, "clusters", clusters)
}

func ClusterShow(rw http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["cluster"]

	cluster, err := models.GetCluster(name)

	if err != nil {
		helpers.Error(nil, err)
		RenderError(rw, err)
		return
	}

	RenderTemplate(rw, "cluster", cluster)
}

func ClusterCreate(rw http.ResponseWriter, r *http.Request) {
	name := GetForm(r, "name")
	size := GetForm(r, "size")
	count := GetForm(r, "count")
	key := GetForm(r, "key")

	cluster := &models.Cluster{
		AvailabilityZones: os.Getenv("AWS_AZS"),
		Name:              name,
		Count:             count,
		Key:               key,
		Size:              size,
	}

	err := cluster.Create()

	if err != nil {
		helpers.Error(nil, err)
		RenderError(rw, err)
		return
	}

	Redirect(rw, r, "/clusters")
}

func ClusterDelete(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["cluster"]

	cluster, err := models.GetCluster(name)

	if err != nil {
		RenderError(rw, err)
		return
	}

	err = cluster.Delete()

	if err != nil {
		helpers.Error(nil, err)
		RenderError(rw, err)
		return
	}

	RenderText(rw, "ok")
}
