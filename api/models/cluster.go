package models

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/ddollar/logger"
)

var DEPLOYMENT_TIMEOUT = 10 * time.Minute

func ClusterIsConverged() (bool, error) {
	var log = logger.New("ns=ClusterIsConverged")

	lsres, err := ECS().ListServices(&ecs.ListServicesInput{
		Cluster: aws.String(os.Getenv("CLUSTER")),
	})

	if err != nil {
		log.Log("at=ListServices err=%q", err)
		return false, err
	}

	dsres, err := ECS().DescribeServices(&ecs.DescribeServicesInput{
		Cluster:  aws.String(os.Getenv("CLUSTER")),
		Services: lsres.ServiceArns,
	})

	if err != nil {
		log.Log("at=DescribeServices err=%q", err)
		return false, err
	}

	for i := 0; i < len(dsres.Services); i++ {
		s := dsres.Services[i]

		// ideal case
		if len(s.Deployments) == 1 &&
			*s.PendingCount == 0 &&
			*s.RunningCount == *s.DesiredCount &&
			strings.HasSuffix(*s.Events[0].Message, "has reached a steady state.") {
			continue
		}

		// service has a wedged deployment, but was manually updated to run 0 tasks
		if *s.DesiredCount == 0 {
			continue
		}

		return false, nil
	}

	return true, nil
}

func GetAppServices(app string) ([]*ecs.Service, error) {
	services := []*ecs.Service{}

	resources, err := ListResources(app)

	if err != nil {
		return services, err
	}

	arns := []*string{}

	for _, r := range resources {
		if r.Type == "Custom::ECSService" {
			arns = append(arns, aws.String(r.Id))
		}
	}

	dres, err := ECS().DescribeServices(&ecs.DescribeServicesInput{
		Cluster:  aws.String(os.Getenv("CLUSTER")),
		Services: arns,
	})

	if err != nil {
		return services, err
	}

	return dres.Services, nil
}

func GetClusterServiceEvents(since time.Time) ([]*ecs.ServiceEvent, error) {
	var log = logger.New("ns=GetClusterServices")

	events := []*ecs.ServiceEvent{}

	lsres, err := ECS().ListServices(&ecs.ListServicesInput{
		Cluster: aws.String(os.Getenv("CLUSTER")),
	})

	if err != nil {
		log.Log("at=ListServices err=%q", err)
		return events, err
	}

	dsres, err := ECS().DescribeServices(&ecs.DescribeServicesInput{
		Cluster:  aws.String(os.Getenv("CLUSTER")),
		Services: lsres.ServiceArns,
	})

	if err != nil {
		log.Log("at=DescribeServices err=%q", err)
		return events, err
	}

	for i := 0; i < len(dsres.Services); i++ {
		s := dsres.Services[i]
		for j := 0; j < len(s.Events); j++ {
			event := s.Events[j]

			if event.CreatedAt.After(since) {
				events = append(events, event)
			}
		}
	}

	return events, nil
}

func ClusterHasCapacityWarning(events []*ecs.ServiceEvent) bool {
	for i := 0; i < len(events); i++ {
		if strings.HasSuffix(*events[i].Message, "see the Troubleshooting section of the Amazon ECS Developer Guide.") {
			return true
		}
	}

	return false
}

// Determine the deployment state based on the state of all the services.
func AppDeploymentState(serviceStates []string) string {
	severity := map[string]int{
		"finished": 0,
		"pending":  1,
		"warning":  2,
		"timeout":  3,
	}

	max := 0
	state := "finished"

	for i := 0; i < len(serviceStates); i++ {
		s := serviceStates[i]
		if severity[s] > max {
			max = severity[s]
			state = s
		}
	}

	return state
}

// Determine the deployment state based on the events that occurred between
// the Deployment.StartedAt and now. For testing purposes take an optional
// time to compare to.
func ServiceDeploymentState(s *ecs.Service, at ...time.Time) string {
	now := time.Now()

	if len(at) > 0 {
		now = at[0]
	}

	// get latest deployment, event, message
	deployment := s.Deployments[0]
	event := s.Events[0]
	message := *event.Message

	fmt.Printf("ServiceDeploymentState message=%q event.CreatedAt=%q deploy.CreatedAt=%q now=%q\n", message, event.CreatedAt, deployment.CreatedAt, now)

	window := now.Add(-DEPLOYMENT_TIMEOUT)

	if deployment.CreatedAt.Before(window) {
		return "timeout"
	}

	if deployment.CreatedAt.After(*event.CreatedAt) {
		return "pending"
	}

	if strings.HasSuffix(message, "reached a steady state.") {
		return "finished"
	}

	if strings.HasSuffix(message, "see the Troubleshooting section of the Amazon ECS Developer Guide.") {
		return "warning"
	}

	return "pending"
}

func ServicesDeploymentStates(services []*ecs.Service) []string {
	states := []string{}

	for i := 0; i < len(services); i++ {
		states = append(states, ServiceDeploymentState(services[i]))
	}

	return states
}
