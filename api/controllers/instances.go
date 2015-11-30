package controllers

import (
	"net/http"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ec2"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/api/models"
)

func init() {
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
