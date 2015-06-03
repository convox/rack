package controllers

import (
	"net/http"

	"github.com/convox/kernel/helpers"
	"github.com/convox/kernel/models"
)

func UpdateInitiate(rw http.ResponseWriter, r *http.Request) {
	err := models.KernelUpdate()

	if err != nil {
		helpers.Error(nil, err)
		RenderError(rw, err)
		return
	}

	Redirect(rw, r, r.Referer())
}
