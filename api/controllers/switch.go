package controllers

import "net/http"

func Switch(w http.ResponseWriter, r *http.Request) error {
	response := map[string]string{
		"source": "rack",
	}
	return RenderJson(w, response)
}
