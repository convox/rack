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
	"net/http"

	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/structs"
)

func IndexDiff(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	var index structs.Index

	err := json.Unmarshal([]byte(r.FormValue("index")), &index)
	if err != nil {
		return httperr.Server(err)
	}

	missing, err := Provider.IndexDiff(&index)
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

		switch header.Typeflag {
		case tar.TypeReg:
			buf := &bytes.Buffer{}
			io.Copy(buf, tr)

			hash := sha256.Sum256(buf.Bytes())

			if header.Name != hex.EncodeToString(hash[:]) {
				return httperr.New(403, fmt.Errorf("invalid hash"))
			}

			if err := Provider.IndexUpload(header.Name, buf.Bytes()); err != nil {
				return httperr.Server(err)
			}
		}
	}

	return RenderSuccess(rw)
}
