package models_test

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/private/protocol/json/jsonutil"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ecs"

	"github.com/convox/rack/api/models"
	"github.com/convox/rack/test"
	"github.com/stretchr/testify/assert"
)

func TestClusterIsConverged(t *testing.T) {
	os.Setenv("RACK", "convox-test")
	os.Setenv("CLUSTER", "convox-test")

	stubAws := test.StubAws(
		test.HttpdListServicesCycle(),
		test.HttpdDescribeServicesCycle(),
	)
	defer stubAws.Close()

	converged, err := models.ClusterIsConverged()
	assert.Nil(t, err)
	assert.True(t, converged)
}

func TestGetServices(t *testing.T) {
	os.Setenv("RACK", "convox-test")
	os.Setenv("CLUSTER", "convox-test")

	aws := test.StubAws(
		test.HttpdDescribeStackResourcesCycle(),
		test.HttpdDescribeServicesCycle(),
	)
	defer aws.Close()

	services, err := models.GetAppServices("httpd")
	assert.Nil(t, err)
	assert.Equal(t, 1, len(services))

	s := services[0]
	assert.Equal(t, "arn:aws:ecs:us-west-2:901416387788:service/httpd-web-SRZPVERKQOL", *s.ServiceArn)
}

func TestServiceDeploymentState(t *testing.T) {
	out := ecs.DescribeServicesOutput{}
	err := jsonutil.UnmarshalJSON(&out, bytes.NewBufferString(test.HttpdDescribeServicesResponse()))
	assert.Nil(t, err)

	s := out.Services[0]
	eventAt := *s.Events[0].CreatedAt

	// final event is "(service httpd-web-SRZPVERKQOL) has reached a steady state."
	assert.Equal(t, 4, len(s.Events))
	assert.Equal(t, "finished", models.ServiceDeploymentState(s, eventAt))

	// shift current event back to "(service httpd-web-SRZPVERKQOL) registered 1 instances in (elb httpd)"
	s.Events = s.Events[1:]
	assert.Equal(t, 3, len(s.Events))
	assert.Equal(t, "pending", models.ServiceDeploymentState(s, eventAt))

	// unshift a scheduler warning
	s.Events = append([]*ecs.ServiceEvent{
		&ecs.ServiceEvent{
			CreatedAt: aws.Time(eventAt),
			Message:   aws.String("service httpd-web-SRZPVERKQOL was unable to place a task because no container instance met all of its requirements. The closest matching container-instance b1a73168-f8a6-4ed9-b69e-94adc7a0f1e0 has insufficient memory available. For more information, see the Troubleshooting section of the Amazon ECS Developer Guide."),
		},
	}, s.Events...)
	assert.Equal(t, 4, len(s.Events))
	assert.Equal(t, "warning", models.ServiceDeploymentState(s, eventAt))

	// unshift a Deployment that started after the last event
	newDeployAt := eventAt.Add(10 * time.Second)
	s.Deployments = append([]*ecs.Deployment{
		&ecs.Deployment{
			Status:    aws.String("PRIMARY"),
			CreatedAt: aws.Time(newDeployAt),
		},
	}, s.Deployments...)

	assert.Equal(t, "pending", models.ServiceDeploymentState(s, newDeployAt))

	// compare deployment start to now which is > 10m after latest event
	assert.Equal(t, "pending", models.ServiceDeploymentState(s, newDeployAt.Add(models.DEPLOYMENT_TIMEOUT)))
	assert.Equal(t, "timeout", models.ServiceDeploymentState(s, newDeployAt.Add(models.DEPLOYMENT_TIMEOUT+1*time.Second)))
}

func TestAppDeploymentState(t *testing.T) {
	assert.Equal(t, "finished", models.AppDeploymentState([]string{"finished", "finished"}))
	assert.Equal(t, "pending", models.AppDeploymentState([]string{"finished", "pending"}))
	assert.Equal(t, "warning", models.AppDeploymentState([]string{"finished", "warning"}))
	assert.Equal(t, "timeout", models.AppDeploymentState([]string{"finished", "timeout"}))

	assert.Equal(t, "pending", models.AppDeploymentState([]string{"pending", "pending"}))
	assert.Equal(t, "warning", models.AppDeploymentState([]string{"pending", "warning"}))
	assert.Equal(t, "timeout", models.AppDeploymentState([]string{"pending", "timeout"}))

	assert.Equal(t, "warning", models.AppDeploymentState([]string{"warning", "warning"}))
	assert.Equal(t, "timeout", models.AppDeploymentState([]string{"warning", "timeout"}))

	assert.Equal(t, "timeout", models.AppDeploymentState([]string{"timeout", "timeout"}))
}

func TestClusterCapacityEvents(t *testing.T) {
	os.Setenv("RACK", "convox-test")
	os.Setenv("CLUSTER", "convox-test")

	stubAws := test.StubAws(
		test.HttpdListServicesCycle(),
		test.HttpdDescribeServicesCycle(),

		test.HttpdListServicesCycle(),
		test.HttpdDescribeServicesCycle(),
	)
	defer stubAws.Close()

	events, err := models.GetClusterServiceEvents(time.Unix(0, 0))
	assert.Nil(t, err)
	assert.Equal(t, 4, len(events))

	events, err = models.GetClusterServiceEvents(time.Unix(1450120333, 0)) // just before last event "createdAt": 1450120334.038
	assert.Nil(t, err)
	assert.Equal(t, 1, len(events))

	assert.False(t, models.ClusterHasCapacityWarning(events))

	// unshift a scheduler warning
	events = append([]*ecs.ServiceEvent{
		&ecs.ServiceEvent{
			Message: aws.String("service httpd-web-SRZPVERKQOL was unable to place a task because no container instance met all of its requirements. The closest matching container-instance b1a73168-f8a6-4ed9-b69e-94adc7a0f1e0 has insufficient memory available. For more information, see the Troubleshooting section of the Amazon ECS Developer Guide."),
		},
	}, events...)

	assert.True(t, models.ClusterHasCapacityWarning(events))
}
