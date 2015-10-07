package controllers

import "net/http"

func Switch(w http.ResponseWriter, r *http.Request) *HttpError {
	response := map[string]string{
		"source": "rack",
	}

	return RenderJson(w, response)
}
