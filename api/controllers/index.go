package controllers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/api/provider"
	"github.com/convox/rack/api/structs"
)

func IndexDiff(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	var index structs.Index

	err := json.Unmarshal([]byte(r.FormValue("index")), &index)
	if err != nil {
		return httperr.Server(err)
	}

	missing, err := provider.IndexDiff(&index)
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, missing)
}

func IndexUpload(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	hash := mux.Vars(r)["hash"]

	file, _, err := r.FormFile("data")
	if err != nil {
		return httperr.Server(err)
	}

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return httperr.Server(err)
	}

	sum := sha256.Sum256(data)

	if hash != hex.EncodeToString(sum[:]) {
		return httperr.New(403, fmt.Errorf("invalid hash"))
	}

	err = provider.IndexUpload(hash, data)
	if err != nil {
		return httperr.Server(err)
	}

	return RenderSuccess(rw)
}
