package controllers

import (
	"net/http"

	"github.com/convox/kernel/models"
)

func UpdateInitiate(rw http.ResponseWriter, r *http.Request) {
	err := models.KernelUpdate()

	if err != nil {
		RenderError(rw, err)
		return
	}

	Redirect(rw, r, r.Referer())
}
