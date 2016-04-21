package controllers

import (
	"net/http"

	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/api/provider"
	"github.com/gorilla/mux"
)

func CertificateCreate(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	pub := r.FormValue("pub")
	key := r.FormValue("key")
	chain := r.FormValue("chain")

	cert, err := provider.CertificateCreate(pub, key, chain)

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, cert)
}

func CertificateDelete(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	id := mux.Vars(r)["id"]

	err := provider.CertificateDelete(id)

	if err != nil {
		return httperr.Server(err)
	}

	return RenderSuccess(rw)
}

func CertificateList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	certs, err := provider.CertificateList()

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, certs)
}
