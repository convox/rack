package controllers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/convox/rack/api/models"
	"github.com/gorilla/mux"
	"golang.org/x/net/websocket"
)

func ServiceList(rw http.ResponseWriter, r *http.Request) error {
	services, err := models.ListServices()

	if err != nil {
		return err
	}

	return RenderJson(rw, services)
}

func ServiceShow(rw http.ResponseWriter, r *http.Request) error {
	service := mux.Vars(r)["service"]

	s, err := models.GetService(service)

	if awsError(err) == "ValidationError" {
		return RenderNotFound(rw, fmt.Sprintf("no such service: %s", service))
	}

	if err != nil {
		return err
	}

	return RenderJson(rw, s)
}

func ServiceCreate(rw http.ResponseWriter, r *http.Request) error {
	name := GetForm(r, "name")
	t := GetForm(r, "type")

	service := &models.Service{
		Name: name,
		Type: t,
	}

	err := service.Create()

	if err != nil && strings.HasSuffix(err.Error(), "not found") {
		return RenderForbidden(rw, fmt.Sprintf("invalid service type: %s", t))
	}

	if err != nil && awsError(err) == "ValidationError" {
		return RenderForbidden(rw, fmt.Sprintf("invalid service name: %s", name))
	}

	if err != nil {
		return err
	}

	service, err = models.GetService(name)

	if err != nil {
		return err
	}

	return RenderJson(rw, service)
}

func ServiceDelete(rw http.ResponseWriter, r *http.Request) error {
	service := mux.Vars(r)["service"]

	s, err := models.GetService(service)

	if awsError(err) == "ValidationError" {
		return RenderNotFound(rw, fmt.Sprintf("no such service: %s", service))
	}

	if err != nil {
		return err
	}

	err = s.Delete()

	if err != nil {
		return err
	}

	s, err = models.GetService(service)

	if err != nil {
		return err
	}

	return RenderJson(rw, s)
}

func ServiceLogs(ws *websocket.Conn) *HttpError {
	service := mux.Vars(ws.Request())["service"]

	s, err := models.GetService(service)

	if err != nil {
		return ServerError(err)
	}

	logs := make(chan []byte)
	done := make(chan bool)

	s.SubscribeLogs(logs, done)

	for data := range logs {
		ws.Write(data)
	}

	return nil
}
