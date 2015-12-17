package workers

import (
	"fmt"
	"os"
	"time"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/ddollar/logger"
	"github.com/convox/rack/api/models"
)

// Monitor all ECS Cluster Service Events and notify on capacity issues.
// Start by looking at events 5m before rack boot, then periodically poll
// for new events.
func StartServicesCapacity() {
	start := time.Now().Add(-5 * time.Minute)

	next := checkCapacity(start)

	for _ = range time.Tick(1 * time.Minute) {
		next = checkCapacity(next)
	}
}

func checkCapacity(since time.Time) time.Time {
	log := logger.New("ns=services_monitor")

	next := time.Now()

	events, err := models.GetClusterServiceEvents(since)

	if err != nil {
		log.Log("fn=GetClusterServiceEvents since=%q err=%q", since, err)
		return since
	}

	log.Log("fn=GetClusterServiceEvents since=%q events=%d", since, len(events))

	if models.ClusterHasCapacityWarning(events) {
		models.NotifyError("rack:capacity", fmt.Errorf("ECS reports a recent capacity issue"), map[string]string{"rack": os.Getenv("RACK")})
	}

	return next
}
