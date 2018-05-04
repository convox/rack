package controllers

import (
	"net/http"

	"github.com/convox/rack/api/httperr"
)

func RackList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	s, err := Provider.SystemGet()
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, []map[string]string{
		{"name": s.Name, "status": "running"},
	})
}
