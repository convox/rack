package controllers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/convox/rack/api/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws/awserr"
	"github.com/convox/rack/api/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/convox/rack/api/models"
)

func FormationList(rw http.ResponseWriter, r *http.Request) error {
	app := mux.Vars(r)["app"]

	_, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		return RenderNotFound(rw, fmt.Sprintf("no such app: %s", app))
	}

	formation, err := models.ListFormation(app)

	if err != nil {
		return err
	}

	return RenderJson(rw, formation)
}

func FormationSet(rw http.ResponseWriter, r *http.Request) error {
	vars := mux.Vars(r)
	app := vars["app"]
	process := vars["process"]
	count := GetForm(r, "count")
	memory := GetForm(r, "memory")

	_, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		return RenderNotFound(rw, fmt.Sprintf("no such app: %s", app))
	}

	err = models.SetFormation(app, process, count, memory)

	if ae, ok := err.(awserr.Error); ok {
		if ae.Code() == "ValidationError" {
			switch {
			case strings.Index(ae.Error(), "No updates are to be performed") > -1:
				return fmt.Errorf("no updates are to be performed: %s", app)
			case strings.Index(ae.Error(), "can not be updated") > -1:
				return fmt.Errorf("app is already updating: %s", app)
			}
		}
	}

	if err != nil {
		return err
	}

	return RenderSuccess(rw)
}
