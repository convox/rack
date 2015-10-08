package controllers

import (
	"net/http"

	"github.com/convox/rack/api/httperr"
)

func Switch(w http.ResponseWriter, r *http.Request) *httperr.Error {
	response := map[string]string{
		"source": "rack",
	}

	return RenderJson(w, response)
}
