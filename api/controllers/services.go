package controllers

import (
	"net/http"
	"strings"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/convox/rack/Godeps/_workspace/src/golang.org/x/net/websocket"
	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/api/models"
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

	switch t {
	case "papertrail":
		err = service.CreatePapertrail()
	case "webhook":
		err = service.CreateWebhook()
	default:
		err = service.CreateDatastore()
	}

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

func ServiceLogs(ws *websocket.Conn) *httperr.Error {
	service := mux.Vars(ws.Request())["service"]

	s, err := models.GetService(service)

	if err != nil {
		return httperr.Server(err)
	}

	logs := make(chan []byte)
	done := make(chan bool)

	s.SubscribeLogs(logs, done)

	go signalWsClose(ws, done)

	for data := range logs {
		ws.Write(data)
	}

	return nil
}
