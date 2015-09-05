package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/convox/kernel/models"
)

func init() {
}

func RackShow(rw http.ResponseWriter, r *http.Request) error {
	rack, err := models.GetRack()

	if awsError(err) == "ValidationError" {
		return RenderNotFound(rw, fmt.Sprintf("no such stack: %s", rack))
	}

	if err != nil {
		return err
	}

	return RenderJson(rw, rack)
}

func RackUpdate(rw http.ResponseWriter, r *http.Request) error {
	rack, err := models.GetRack()

	if err != nil {
		return err
	}

	if count := GetForm(r, "count"); count != "" {
		count, err := strconv.Atoi(count)

		if err != nil {
			return err
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
			return RenderForbidden(rw, fmt.Sprintf("no system updates are to be performed."))
		case strings.Index(err.Error(), "can not be updated") > -1:
			return RenderForbidden(rw, fmt.Sprintf("system is already updating."))
		}
	}

	if err != nil {
		return err
	}

	rack, err = models.GetRack()

	if err != nil {
		return err
	}

	return RenderJson(rw, rack)
}
