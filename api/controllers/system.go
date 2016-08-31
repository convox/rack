package controllers

import (
	"net/http"
	"os"
	"strconv"

	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/api/models"
)

func SystemShow(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	rack, err := models.Provider().SystemGet()
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, rack)
}

func SystemUpdate(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	rack, err := models.Provider().SystemGet()
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
		case os.Getenv("AUTOSCALE") == "true":
			return httperr.Errorf(403, "scaling count prohibited when autoscale enabled")
		case c == -1:
			// -1 indicates no change
		case c <= 1:
			return httperr.Errorf(403, "count must be greater than 1")
		default:
			rack.Count = c
		}
	}

	if t := GetForm(r, "type"); t != "" {
		rack.Type = t
	}

	if v := GetForm(r, "version"); v != "" {
		rack.Version = v
	}

	err = models.Provider().SystemSave(*rack)
	if err != nil {
		return httperr.Server(err)
	}

	// models.NotifySuccess("rack:update", notifyData)

	return RenderJson(rw, rack)
}

func SystemCapacity(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	capacity, err := models.Provider().CapacityGet()
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, capacity)
}

// SystemReleases lists the latest releases of the rack
func SystemReleases(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	releases, err := models.Provider().SystemReleases()
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, releases)
}
