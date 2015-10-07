package controllers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/convox/rack/api/models"
)

func init() {
}

func SystemShow(rw http.ResponseWriter, r *http.Request) *HttpError {
	rack, err := models.GetSystem()

	if awsError(err) == "ValidationError" {
		return HttpErrorf(404, "no such stack: %s", rack)
	}

	if err != nil {
		return ServerError(err)
	}

	return RenderJson(rw, rack)
}

func SystemUpdate(rw http.ResponseWriter, r *http.Request) *HttpError {
	rack, err := models.GetSystem()

	if err != nil {
		return ServerError(err)
	}

	if count := GetForm(r, "count"); count != "" {
		count, err := strconv.Atoi(count)

		if err != nil {
			return ServerError(err)
		}

		rack.Count = count
	}

	if t := GetForm(r, "type"); t != "" {
		rack.Type = t
	}

	if version := GetForm(r, "version"); version != "" {
		rack.Version = version
	}

	err = rack.Save()

	if awsError(err) == "ValidationError" {
		switch {
		case strings.Index(err.Error(), "No updates are to be performed") > -1:
			return HttpErrorf(403, "no system updates are to be performed")
		case strings.Index(err.Error(), "can not be updated") > -1:
			return HttpErrorf(403, "system is already updating")
		}
	}

	if err != nil {
		return ServerError(err)
	}

	rack, err = models.GetSystem()

	if err != nil {
		return ServerError(err)
	}

	return RenderJson(rw, rack)
}
