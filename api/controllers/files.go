package controllers

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/convox/rack/api/httperr"
	"github.com/gorilla/mux"
)

func FilesDelete(w http.ResponseWriter, r *http.Request) *httperr.Error {
	v := mux.Vars(r)
	app := v["app"]
	pid := v["process"]

	if strings.TrimSpace(pid) == "" {
		return httperr.Server(fmt.Errorf("must specify a pid"))
	}

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return httperr.Server(err)
	}

	uv, err := url.ParseQuery(string(data))
	if err != nil {
		return httperr.Server(err)
	}

	files := strings.Split(uv.Get("files"), ",")

	if len(files) == 0 {
		return httperr.Server(fmt.Errorf("must specify at least one file"))
	}

	if err := Provider.FilesDelete(app, pid, files); err != nil {
		return httperr.Server(err)
	}

	return nil
}

func FilesUpload(w http.ResponseWriter, r *http.Request) *httperr.Error {
	v := mux.Vars(r)
	app := v["app"]
	pid := v["process"]

	if err := Provider.FilesUpload(app, pid, r.Body); err != nil {
		return httperr.Server(err)
	}

	return nil
}
