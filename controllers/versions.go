package controllers

import (
	"net/http"
	"os"
)

func VersionGet(rw http.ResponseWriter, r *http.Request) {
	RenderText(rw, os.Getenv("RELEASE"))
}
