package controllers

import (
	"net/http"
	"strings"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/awserr"
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
	err := r.ParseForm()
	if err != nil {
		return httperr.Server(err)
	}

	// get the last set value for all form values
	// ie:  foo=1&foo=2  sets foo to "2"
	params := make(map[string]string)
	for key, values := range r.Form {
		val := values[len(values)-1]
		params[key] = val
	}
	name := params["name"]
	delete(params, "name")
	kind := params["type"]
	delete(params, "type")

	// Early check for unbound service only.
	service, err := models.GetServiceUnbound(name)

	if err != nil {
		return httperr.Errorf(403, "there is already a legacy service named %s (%s). We recommend you delete this service and create it again.", name, service.Status)
	}

	if awsError(err) == "ValidationError" {
		// If unbound check fails this will result in a bound service.
		service = &models.Service{
			Name:       name,
			Type:       kind,
			Parameters: models.CFParams(params),
		}
	}

	err = service.Create()

	if err != nil && strings.HasSuffix(err.Error(), "not found") {
		return httperr.Errorf(403, "invalid service type: %s", kind)
	}

	if err != nil && awsError(err) == "ValidationError" {
		e := err.(awserr.Error)
		return httperr.Errorf(403, convoxifyCloudformationError(e.Message()))
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

func convoxifyCloudformationError(msg string) string {
	newMsg := strings.Replace(msg, "do not exist in the template", "are not supported by this service", 1)
	newMsg = strings.Replace(newMsg, "Parameters:", "Options:", 1)
	return newMsg
}
