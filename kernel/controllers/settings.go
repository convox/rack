package controllers

import "net/http"

func init() {
	RegisterTemplate("settings", "layout", "settings")
}

func Settings(rw http.ResponseWriter, r *http.Request) {
	RenderTemplate(rw, "settings", nil)
}
