package controllers

import (
	"net/http"
	"strconv"

	"github.com/convox/rack/api/httperr"
	"github.com/gorilla/mux"
)

func FormationList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	app := mux.Vars(r)["app"]

	formation, err := Provider.FormationList(app)
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, formation)
}

func FormationSet(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)
	app := vars["app"]
	process := vars["process"]

	pf, err := Provider.FormationGet(app, process)
	if err != nil {
		return httperr.Server(err)
	}

	// update based on form input
	if cc := GetForm(r, "count"); cc != "" {
		c, err := strconv.Atoi(cc)
		if err != nil {
			return httperr.Errorf(403, "count must be numeric")
		}

		switch {
		// critical fix: old clients default to count=-1 for "no change"
		// assert a minimum client version before setting count=-1 which now deletes a service / ELB
		case r.Header.Get("Version") < "20160602213113" && c == -1:
		// backwards compatibility: other old clients use count=-2 for "no change"
		case c == -2:
		default:
			pf.Count = c
		}
	}

	if cc := GetForm(r, "cpu"); cc != "" {
		c, err := strconv.Atoi(cc)
		if err != nil {
			return httperr.Errorf(403, "cpu must be numeric")
		}

		switch {
		// backwards compatibility: other old clients use cpu=-1 for "no change"
		case c == -1:
		default:
			pf.CPU = c
		}
	}

	if mm := GetForm(r, "memory"); mm != "" {
		m, err := strconv.Atoi(mm)
		if err != nil {
			return httperr.Errorf(403, "memory must be numeric")
		}

		switch {
		// backwards compatibility: other old clients use memory=-1 or memory=0 for "no change"
		case m == -1 || m == 0:
		default:
			pf.Memory = m
		}
	}

	err = Provider.FormationSave(app, pf)
	if err != nil {
		return httperr.Server(err)
	}

	return RenderSuccess(rw)
}
