package controllers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/api/models"
	"github.com/gorilla/mux"
	"golang.org/x/net/websocket"
)

func ServiceList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	services, err := models.ListServices()

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, services)
}

func ServiceShow(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	service := mux.Vars(r)["service"]

	s, err := models.GetService(service)

	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such service: %s", service)
	}

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, s)
}

func ServiceCreate(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	name := GetForm(r, "name")
	t := GetForm(r, "type")
	url := GetForm(r, "url")

	service := &models.Service{
		Name: name,
		Type: t,
		URL:  url,
	}

	var err error

	if t == "papertrail" {
		err = service.CreatePapertrail()
	} else {
		err = service.Create()
	}

	fmt.Printf("%+v\n", err)

	if err != nil && strings.HasSuffix(err.Error(), "not found") {
		return httperr.Errorf(403, "invalid service type: %s", t)
	}

	if err != nil && awsError(err) == "ValidationError" {
		return httperr.Errorf(403, "invalid service name: %s", name)
	}

	if err != nil {
		return httperr.Server(err)
	}

	service, err = models.GetService(name)

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, service)
}

func ServiceDelete(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	service := mux.Vars(r)["service"]

	s, err := models.GetService(service)

	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such service: %s", service)
	}

	if err != nil {
		return httperr.Server(err)
	}

	err = s.Delete()

	if err != nil {
		return httperr.Server(err)
	}

	s, err = models.GetService(service)

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, s)
}

func ServiceLink(rw http.ResponseWriter, r *http.Request) error {
	service := mux.Vars(r)["service"]

	s, err := models.GetService(service)

	if awsError(err) == "ValidationError" {
		return RenderNotFound(rw, fmt.Sprintf("no such service: %s", service))
	}

	if err != nil {
		return err
	}

	if s.Status != "running" {
		return RenderForbidden(rw, fmt.Sprintf("can not link service with status: %s", s.Status))
	}

	if s.Type != "papertrail" {
		return RenderForbidden(rw, fmt.Sprintf("linking is not yet implemented for service type: %s", s.Type))
	}

	app := GetForm(r, "app")

	a, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		return RenderNotFound(rw, fmt.Sprintf("no such app: %s", app))
	}

	if err != nil {
		return err
	}

	err = s.LinkPapertrail(*a)

	if err != nil {
		return err
	}

	return RenderJson(rw, s)
}

func ServiceUnlink(rw http.ResponseWriter, r *http.Request) error {
	app := mux.Vars(r)["app"]
	service := mux.Vars(r)["service"]

	s, err := models.GetService(service)

	if awsError(err) == "ValidationError" {
		return RenderNotFound(rw, fmt.Sprintf("no such service: %s", service))
	}

	if err != nil {
		return err
	}

	if s.Status != "running" {
		return RenderForbidden(rw, fmt.Sprintf("can not unlink service with status: %s", s.Status))
	}

	if s.Type != "papertrail" {
		return RenderForbidden(rw, fmt.Sprintf("unlinking is not yet implemented for service type: %s", s.Type))
	}

	a, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		return RenderNotFound(rw, fmt.Sprintf("no such app: %s", app))
	}

	if err != nil {
		return err
	}

	err = s.UnlinkPapertrail(*a)

	if err != nil {
		return err
	}

	return RenderJson(rw, s)
}

func ServiceLogs(ws *websocket.Conn) *httperr.Error {
	service := mux.Vars(ws.Request())["service"]

	s, err := models.GetService(service)

	if err != nil {
		return httperr.Server(err)
	}

	logs := make(chan []byte)
	done := make(chan bool)

	s.SubscribeLogs(logs, done)

	for data := range logs {
		ws.Write(data)
	}

	return nil
}
