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
	converged := true

	converged, lastEvent := checkConvergence(converged, time.Now())
	// lastEvent := checkCapacity(time.Now())

	for _ = range time.Tick(1 * time.Minute) {
		converged, lastEvent = checkConvergence(converged, *lastEvent.CreatedAt)
	}
}

func checkConvergence(lastConverged bool, lastEventAt time.Time) (bool, ecs.ServiceEvent) {
	log := logger.New("ns=services_monitor")

	services, err := models.ClusterServices()

	if err != nil {
		log.Log("fn=ClusterServices err=%q", lastEventAt, err)

		return lastConverged, ecs.ServiceEvent{
			CreatedAt: aws.Time(lastEventAt),
		}
	}

	converged := services.IsConverged()

	if converged != lastConverged {
		models.NotifySuccess("rack:converge", map[string]string{
			"rack":      os.Getenv("RACK"),
			"converged": fmt.Sprintf("%t", converged),
		})
	}

	events := services.EventsSince(lastEventAt)

	log.Log("fn=EventsSince lastEventAt=%q events=%d", lastEventAt, len(events))

	if events.HasCapacityWarning() {
		models.NotifyError("rack:capacity", fmt.Errorf("ECS reports a recent capacity issue"), map[string]string{"rack": os.Getenv("RACK")})
	}

	return converged, services.LastEvent()
}
