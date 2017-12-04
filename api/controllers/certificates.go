package controllers

import (
	"net/http"
	"sort"
	"strings"

	"github.com/convox/rack/api/httperr"
	"github.com/gorilla/mux"
)

func CertificateCreate(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	pub := r.FormValue("public")
	key := r.FormValue("private")
	chain := r.FormValue("chain")

	cert, err := Provider.CertificateCreate(pub, key, chain)
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, cert)
}

func CertificateDelete(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	id := mux.Vars(r)["id"]

	err := Provider.CertificateDelete(id)

	if err != nil {
		return httperr.Server(err)
	}

	return RenderSuccess(rw)
}

func CertificateGenerate(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	domains := strings.Split(r.FormValue("domains"), ",")

	cert, err := Provider.CertificateGenerate(domains)

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, cert)
}

func CertificateList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	certs, err := Provider.CertificateList()

	if err != nil {
		return httperr.Server(err)
	}

	sort.Sort(certs)

	return RenderJson(rw, certs)
}
