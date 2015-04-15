package controllers

import (
	"net/http"

	"github.com/convox/kernel/models"
)

func init() {
	RegisterTemplate("settings", "layout", "settings")
}

func SettingsList(rw http.ResponseWriter, r *http.Request) {
	github, err := models.SettingGet("github")

	if err != nil {
		RenderError(rw, err)
		return
	}

	heroku, err := models.SettingGet("heroku")

	if err != nil {
		RenderError(rw, err)
		return
	}

	params := map[string]interface{}{
		"Github": github,
		"Heroku": heroku,
	}

	RenderTemplate(rw, "settings", params)
}

func SettingsUpdate(rw http.ResponseWriter, r *http.Request) {
	err := models.SettingSet("github", GetForm(r, "github"))

	if err != nil {
		RenderError(rw, err)
		return
	}

	err = models.SettingSet("heroku", GetForm(r, "heroku"))

	if err != nil {
		RenderError(rw, err)
		return
	}

	Redirect(rw, r, "/settings")
}
