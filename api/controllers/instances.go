package controllers

import (
	"net/http"
	"strconv"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/api/models"
	"github.com/convox/rack/api/provider"
	"github.com/gorilla/mux"
	"golang.org/x/net/websocket"
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
	instances, err := provider.InstanceList()

	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, instances)
}

func InstanceSSH(ws *websocket.Conn) *httperr.Error {
	vars := mux.Vars(ws.Request())
	id := vars["id"]
	cmd := ws.Request().Header.Get("Command")

	term := ws.Request().Header.Get("Terminal")
	var height, width int
	var err error

	if term != "" {
		height, err = strconv.Atoi(ws.Request().Header.Get("Height"))
		if err != nil {
			return httperr.Server(err)
		}
		width, err = strconv.Atoi(ws.Request().Header.Get("Width"))
		if err != nil {
			return httperr.Server(err)
		}
	}

	return httperr.Server(models.InstanceSSH(id, cmd, term, height, width, ws))
}

func InstanceTerminate(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	rack, err := provider.SystemGet()

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
