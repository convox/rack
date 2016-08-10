package controllers

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/api/provider"
	"github.com/convox/rack/api/structs"
	"github.com/gorilla/mux"
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

// IndexUpdate accepts a tarball of changes to the index
func IndexUpdate(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	update, _, err := r.FormFile("update")
	if err != nil {
		return httperr.Server(err)
	}

	gz, err := gzip.NewReader(update)
	if err != nil {
		return httperr.Server(err)
	}

	tr := tar.NewReader(gz)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return httperr.Server(err)
		}

		fmt.Printf("header = %+v\n", header)

		switch header.Typeflag {
		case tar.TypeReg:
			buf := &bytes.Buffer{}
			io.Copy(buf, tr)

			hash := sha256.Sum256(buf.Bytes())

			if header.Name != hex.EncodeToString(hash[:]) {
				return httperr.New(403, fmt.Errorf("invalid hash"))
			}

			if err := provider.IndexUpload(header.Name, buf.Bytes()); err != nil {
				return httperr.Server(err)
			}
		}
	}

	return RenderSuccess(rw)
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
