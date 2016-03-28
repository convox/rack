package controllers

import (
	"net/http"
	"os"

	"github.com/convox/rack/api/httperr"
)

func RackList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	return RenderJson(rw, []map[string]string{
		map[string]string{
			"name":   os.Getenv("RACK"),
			"status": "running",
		},
	})
}
