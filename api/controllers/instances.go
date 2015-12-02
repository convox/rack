package controllers

import (
	"net/http"
	"strconv"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ec2"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/convox/rack/Godeps/_workspace/src/golang.org/x/net/websocket"
	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/api/models"
)

func init() {
}

func InstancesKeyroll(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	err := models.InstanceKeyroll()

	if err != nil {
		return httperr.Server(err)
	}

	return RenderSuccess(rw)
}

func InstancesList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	rack, err := models.GetSystem()

	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such stack: %s", rack)
	}

	if err != nil {
		return httperr.Server(err)
	}

	instances, err := rack.GetInstances()

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, instances)
}

func InstanceSSH(ws *websocket.Conn) *httperr.Error {
	vars := mux.Vars(ws.Request())
	id := vars["id"]
	cmd := ws.Request().Header.Get("Command")
	height, err := strconv.Atoi(ws.Request().Header.Get("Height"))
	if err != nil {
		return httperr.Server(err)
	}
	width, err := strconv.Atoi(ws.Request().Header.Get("Width"))
	if err != nil {
		return httperr.Server(err)
	}

	return httperr.Server(models.InstanceSSH(id, cmd, height, width, ws))
}

func InstanceTerminate(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	rack, err := models.GetSystem()

	if awsError(err) == "ValidationError" {
		return httperr.Errorf(404, "no such stack: %s", rack)
	}

	if err != nil {
		return httperr.Server(err)
	}

	instanceId := mux.Vars(r)["id"]

	_, err = models.EC2().TerminateInstances(&ec2.TerminateInstancesInput{
		InstanceIds: []*string{&instanceId},
	})

	if err != nil {
		return httperr.Server(err)
	}

	return RenderSuccess(rw)
}
