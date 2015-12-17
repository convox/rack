package workers

import (
	"fmt"
	"os"
	"time"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/ddollar/logger"
	"github.com/convox/rack/api/models"
)

// Monitor ECS Cluster for convergence.
// When not converged, notify on capacity issues.
// When re-converged, try to correlate back to a recent service deployment.

func StartServicesCapacity() {
	converged, lastEvent := checkConverged()

	for _ = range time.Tick(1 * time.Minute) {
		converged, lastEvent = monitorConverged(converged, *lastEvent.CreatedAt)
	}
}

// get initial convergence state
func checkConverged() (bool, ecs.ServiceEvent) {
	log := logger.New("ns=services_monitor")

	services, err := models.ClusterServices()

	if err != nil {
		log.Log("fn=checkConverged err=%q", err)

		return true, ecs.ServiceEvent{
			CreatedAt: aws.Time(time.Now()),
		}
	}

	converged := services.IsConverged()
	lastEvent := services.LastEvent()

	log.Log("fn=checkConverged converged=%t lastEventAt=%q", converged, lastEvent.CreatedAt)

	return converged, lastEvent
}

// get latest convergence state, notify on changes
func monitorConverged(lastConverged bool, lastEventAt time.Time) (bool, ecs.ServiceEvent) {
	log := logger.New("ns=services_monitor")

	services, err := models.ClusterServices()

	if err != nil {
		log.Log("fn=monitorConverged err=%q", err)

		return lastConverged, ecs.ServiceEvent{
			CreatedAt: aws.Time(lastEventAt),
		}
	}

	converged := services.IsConverged()
	events := services.EventsSince(lastEventAt)

	log.Log("fn=monitorConverged converged=%t events=%d lastEventAt=%q", converged, len(events), lastEventAt)

	if events.HasCapacityWarning() {
		models.NotifyError("rack:capacity", fmt.Errorf("ECS reports a recent capacity issue"), map[string]string{
			"rack": os.Getenv("RACK"),
		})
	}

	if converged != lastConverged {
		models.NotifySuccess("rack:converge", map[string]string{
			"rack":      os.Getenv("RACK"),
			"converged": fmt.Sprintf("%t", converged),
		})
	}

	return converged, services.LastEvent()
}
