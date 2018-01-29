package controllers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/convox/rack/api/httperr"
	"github.com/gorilla/mux"
)

type SSL struct {
	Certificate string    `json:"certificate"`
	Domain      string    `json:"domain"`
	Expiration  time.Time `json:"expiration"`
	Process     string    `json:"process"`
	Port        int       `json:"port"`
}

type SSLs []SSL

func SSLList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	app := mux.Vars(r)["app"]

	ss, err := Provider.ServiceList(app)
	if err != nil {
		return httperr.Server(err)
	}

	certs, err := Provider.CertificateList()
	if err != nil {
		return httperr.Server(err)
	}

	ssls := SSLs{}

	for _, s := range ss {
		for _, sp := range s.Ports {
			for _, c := range certs {
				if c.Id == sp.Certificate {
					ssls = append(ssls, SSL{
						Certificate: c.Id,
						Domain:      c.Domain,
						Expiration:  c.Expiration,
						Process:     s.Name,
						Port:        sp.Balancer,
					})
				}
			}
		}
	}

	return RenderJson(rw, ssls)
}

func SSLUpdate(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)
	app := vars["app"]
	service := vars["process"]
	port := vars["port"]
	cert := GetForm(r, "id")

	porti, err := strconv.Atoi(port)
	if err != nil {
		return httperr.Errorf(403, "port must be numeric")
	}

	if err := Provider.CertificateApply(app, service, porti, cert); err != nil {
		return httperr.Server(err)
	}

	return RenderSuccess(rw)
}
