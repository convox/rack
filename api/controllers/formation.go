package controllers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/api/models"
	"github.com/gorilla/mux"
)

func FormationList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	app := mux.Vars(r)["app"]

	_, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", app)
	}

	formation, err := models.ListFormation(app)

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, formation)
}

func FormationSet(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	vars := mux.Vars(r)
	app := vars["app"]
	process := vars["process"]

	_, err := models.GetApp(app)
	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such app: %s", app)
	}

	// initialize to invalid values that indicate no change
	var count, memory int64 = -2, -1

	// update based on form input
	if cc := GetForm(r, "count"); cc != "" {
		if c, err := strconv.ParseInt(cc, 10, 64); err != nil {
			return httperr.Errorf(403, "count must be numeric")
		} else {
			// critical fix: old clients default to count=-1 for "no change"
			// assert a minimum client version before setting count=-1 which now deletes a service / ELB
			if c == -1 && r.Header.Get("Version") < "20160602213113" {
				count = -2
			} else {
				count = c
			}
		}
	}

	if mm := GetForm(r, "memory"); mm != "" {
		if m, err := strconv.ParseInt(mm, 10, 64); err != nil {
			return httperr.Errorf(403, "memory must be numeric")
		} else {
			memory = m
		}
	}

	err = models.SetFormation(app, process, count, memory)
	if ae, ok := err.(awserr.Error); ok {
		if ae.Code() == "ValidationError" {
			switch {
			case strings.Index(ae.Error(), "No updates are to be performed") > -1:
				return httperr.Errorf(403, "no updates are to be performed: %s", app)
			case strings.Index(ae.Error(), "can not be updated") > -1:
				return httperr.Errorf(403, "app is already updating: %s", app)
			}
		}
	}
	if err != nil {
		return httperr.Server(err)
	}

	return RenderSuccess(rw)
}
