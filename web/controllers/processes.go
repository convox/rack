package controllers

import (
	"net/http"

	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/gorilla/mux"

	"github.com/convox/kernel/web/models"
)

func init() {
	RegisterTemplate("process", "layout", "process")
}

func ProcessShow(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	process, err := models.GetProcess(vars["app"], vars["process"])

	if err != nil {
		RenderError(rw, err)
		return
	}

	RenderTemplate(rw, "process", process)
}
