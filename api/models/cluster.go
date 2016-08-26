package models

import (
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
)

type ECSServices []*ecs.Service
type ECSEvents []*ecs.ServiceEvent

var DEPLOYMENT_TIMEOUT = 10 * time.Minute

func ClusterServices() (ECSServices, error) {
	log := Logger.At("ClusterServices").Start()

	services := ECSServices{}

	lsres, err := ECS().ListServices(&ecs.ListServicesInput{
		Cluster: aws.String(os.Getenv("CLUSTER")),
	})

	if err != nil {
		log.Step("ListServices").Error(err)
		return services, err
	}

	dsres, err := ECS().DescribeServices(&ecs.DescribeServicesInput{
		Cluster:  aws.String(os.Getenv("CLUSTER")),
		Services: lsres.ServiceArns,
	})

	if err != nil {
		log.Step("DescribeServices").Error(err)
		return services, err
	}

	for i := 0; i < len(dsres.Services); i++ {
		services = append(services, dsres.Services[i])
	}

	return services, nil
}

func (services ECSServices) IsConverged() bool {
	for i := 0; i < len(services); i++ {
		s := services[i]

		// ideal case for a Service
		if len(s.Deployments) == 1 &&
			*s.PendingCount == 0 &&
			*s.RunningCount == *s.DesiredCount &&
			strings.HasSuffix(*s.Events[0].Message, "has reached a steady state.") {
			continue
		}

		// Service has a wedged deployment, but was manually updated to run 0 tasks
		if *s.DesiredCount == 0 {
			continue
		}

		return false
	}

	return true
}

func (services ECSServices) LastEvent() ecs.ServiceEvent {
	e := ecs.ServiceEvent{
		CreatedAt: aws.Time(time.Unix(0, 0)),
	}

	for i := 0; i < len(services); i++ {
		s := services[i]

		if len(s.Events) > 0 && s.Events[0].CreatedAt.After(*e.CreatedAt) {
			e = *s.Events[0]
		}
	}

	return e
}

func (services ECSServices) EventsSince(since time.Time) ECSEvents {
	events := ECSEvents{}

	for i := 0; i < len(services); i++ {
		s := services[i]

		for j := 0; j < len(s.Events); j++ {
			event := s.Events[j]

			if event.CreatedAt.After(since) {
				events = append(events, event)
			}
		}
	}

	return events
}

func (events ECSEvents) HasCapacityWarning() bool {
	return events.CapacityWarning() != ""
}

var warningSuffix string = "For more information, see the Troubleshooting section of the Amazon ECS Developer Guide."

func (events ECSEvents) CapacityWarning() string {
	for i := 0; i < len(events); i++ {
		message := *events[i].Message
		if strings.HasSuffix(message, warningSuffix) {
			return strings.TrimSpace(strings.TrimSuffix(message, warningSuffix))
		}
	}

	return ""
}
