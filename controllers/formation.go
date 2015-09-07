package controllers

import (
	"fmt"
	"net/http"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/convox/kernel/models"
)

func FormationList(rw http.ResponseWriter, r *http.Request) error {
	app := mux.Vars(r)["app"]

	_, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		return RenderNotFound(rw, fmt.Sprintf("no such app: %s", app))
	}

	formation, err := models.ListFormation(app)

	if err != nil {
		return err
	}

	return RenderJson(rw, formation)
}
