package controllers

import (
	"net/http"
	"strconv"

	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/options"
	"github.com/convox/rack/structs"
	"github.com/gorilla/mux"
)

func FormationList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	app := mux.Vars(r)["app"]

	ss, err := Provider.ServiceList(app)
	if err != nil {
		return httperr.Server(err)
	}

	f := structs.Formation{}

	for _, s := range ss {
		pf := structs.ProcessFormation{
			Balancer: s.Domain,
			Count:    s.Count,
			CPU:      s.Cpu,
			Memory:   s.Memory,
			Name:     s.Name,
			Ports:    []int{},
		}

		for _, p := range s.Ports {
			pf.Ports = append(pf.Ports, p.Balancer)
		}

		f = append(f, pf)
	}

	return RenderJson(rw, f)
}

func FormationSet(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)
	app := vars["app"]
	service := vars["process"]

	opts := structs.ServiceUpdateOptions{}

	if cc := GetForm(r, "count"); cc != "" {
		c, err := strconv.Atoi(cc)
		if err != nil {
			return httperr.Errorf(403, "count must be numeric")
		}

		opts.Count = options.Int(c)
	}

	if cc := GetForm(r, "cpu"); cc != "" {
		c, err := strconv.Atoi(cc)
		if err != nil {
			return httperr.Errorf(403, "cpu must be numeric")
		}

		opts.Cpu = options.Int(c)
	}

	if cc := GetForm(r, "memory"); cc != "" {
		c, err := strconv.Atoi(cc)
		if err != nil {
			return httperr.Errorf(403, "memory must be numeric")
		}

		opts.Memory = options.Int(c)
	}

	if err := Provider.ServiceUpdate(app, service, opts); err != nil {
		return httperr.Server(err)
	}

	return RenderSuccess(rw)
}
