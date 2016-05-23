package controllers

import (
	"net/http"
	"strconv"

	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/api/models"
	"github.com/convox/rack/api/provider"
)

func SystemReleaseList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	rack, err := provider.SystemGet()
	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such stack: %s", rack)
	}
	if err != nil {
		return httperr.Server(err)
	}

	releases, err := provider.ReleaseList(rack.Name)
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, releases)
}

func SystemShow(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	rack, err := provider.SystemGet()

	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such stack: %s", rack)
	}

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, rack)
}

func SystemUpdate(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	rack, err := provider.SystemGet()

	if err != nil {
		return httperr.Server(err)
	}

	notifyData := map[string]string{}

	if count := GetForm(r, "count"); count != "" {
		count, err := strconv.Atoi(count)
		if err != nil {
			return httperr.Errorf(403, "count must be an integer")
		}

		// -1 is an invalid value that indicates no change
		if count != -1 {
			if count <= 1 {
				return httperr.Errorf(403, "count must be greater than 1")
			}

			rack.Count = count

			notifyData["count"] = strconv.Itoa(count)
		}
	}

	if t := GetForm(r, "type"); t != "" {
		rack.Type = t
		notifyData["type"] = t
	}

	if version := GetForm(r, "version"); version != "" {
		rack.Version = version
		notifyData["version"] = version
	}

	if len(notifyData) == 0 {
		return httperr.Errorf(403, "no rack updates specified")
	}

	err = provider.SystemSave(*rack)

	if err != nil {
		return httperr.Server(err)
	}

	rack, err = provider.SystemGet()

	if err != nil {
		return httperr.Server(err)
	}

	models.NotifySuccess("rack:update", notifyData)

	return RenderJson(rw, rack)
}

func SystemCapacity(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	capacity, err := provider.CapacityGet()

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, capacity)
}
