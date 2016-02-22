package models_test

import (
	"os"
	"testing"
	"time"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ecs"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"github.com/convox/rack/api/models"
	"github.com/convox/rack/test"
)

func getClusterServices() (models.ECSServices, error) {
	os.Setenv("RACK", "convox-test")
	os.Setenv("CLUSTER", "convox-test")

	s := test.StubAws(
		test.HttpdListServicesCycle(),
		test.HttpdDescribeServicesCycle(),
	)

	defer s.Close()

	return models.ClusterServices()
}

func TestClusterServices(t *testing.T) {
	services, _ := getClusterServices()

	assert.IsType(t, models.ECSServices{}, services)
	assert.Equal(t, 1, len(services))
}

func TestIsConverged(t *testing.T) {
	services, _ := getClusterServices()

	assert.Equal(t, 1, len(services))
	assert.True(t, services.IsConverged())

	// unshift a new deployment
	s := services[0]
	s.Deployments = append([]*ecs.Deployment{
		&ecs.Deployment{
			Status: aws.String("PRIMARY"),
		},
	}, s.Deployments...)

	assert.False(t, services.IsConverged())
}

func TestLastEvent(t *testing.T) {
	services, _ := getClusterServices()

	event := services.LastEvent()
	assert.Equal(t, "7a8cd970-01ff-4d34-aa34-fa0deff70e48", *event.Id)

	// add another service with a more recent event
	services = append(services, &ecs.Service{
		Events: []*ecs.ServiceEvent{
			&ecs.ServiceEvent{
				Id:        aws.String("ce9dfdaf-864f-4dc2-9581-0b8b0826f0aa"),
				CreatedAt: aws.Time(time.Now()),
			},
		},
	})

	event = services.LastEvent()
	assert.Equal(t, "ce9dfdaf-864f-4dc2-9581-0b8b0826f0aa", *event.Id)
}

func TestEventsSince(t *testing.T) {
	services, _ := getClusterServices()

	events := services.EventsSince(time.Unix(0, 0))
	assert.Equal(t, 4, len(events))

	events = services.EventsSince(time.Unix(1450120333, 0)) // just before last event "createdAt": 1450120334.038
	assert.Equal(t, 1, len(events))
}

func TestEventsCapacityWarning(t *testing.T) {
	services, _ := getClusterServices()

	events := services.EventsSince(time.Unix(0, 0))
	assert.False(t, events.HasCapacityWarning())

	// unshift a scheduler warning
	events = append([]*ecs.ServiceEvent{
		&ecs.ServiceEvent{
			Message: aws.String("service httpd-web-SRZPVERKQOL was unable to place a task because no container instance met all of its requirements. The closest matching container-instance b1a73168-f8a6-4ed9-b69e-94adc7a0f1e0 has insufficient memory available. For more information, see the Troubleshooting section of the Amazon ECS Developer Guide."),
		},
	}, events...)

	assert.True(t, events.HasCapacityWarning())

	assert.Equal(t, "service httpd-web-SRZPVERKQOL was unable to place a task because no container instance met all of its requirements. The closest matching container-instance b1a73168-f8a6-4ed9-b69e-94adc7a0f1e0 has insufficient memory available.", events.CapacityWarning())

}

// FIXME move to provider
// func TestGetAppServices(t *testing.T) {
//   os.Setenv("RACK", "convox-test")
//   os.Setenv("CLUSTER", "convox-test")

//   aws := test.StubAws(
//     test.HttpdDescribeStackResourcesCycle(),
//     test.HttpdDescribeServicesCycle(),
//   )
//   defer aws.Close()

//   services, err := models.GetAppServices("httpd")
//   assert.Nil(t, err)
//   assert.Equal(t, 1, len(services))

//   s := services[0]
//   assert.Equal(t, "arn:aws:ecs:us-west-2:901416387788:service/httpd-web-SRZPVERKQOL", *s.ServiceArn)
// }
