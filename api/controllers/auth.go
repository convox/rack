package controllers

import (
	"net/http"

	"github.com/convox/rack/api/httperr"
)

func init() {
}

// Depends on mount point for auth
// Returns a json packet with a message for a client
// suitable for display to an end user.
func CheckAuth(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	response := map[string]string{
		"message": "Logged in successfully.",
	}

	return RenderJson(rw, response)
}
