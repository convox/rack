package workers

import (
	"time"

	"github.com/cloudflare/cfssl/log"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/ddollar/logger"
	"github.com/convox/rack/api/helpers"
	"github.com/convox/rack/api/models"
)

func StartHeartbeat() {
	log := logger.New("ns=heartbeat")

	defer recoverWith(func(err error) {
		helpers.Error(log, err)
	})

	heartbeat()

	for _ = range time.Tick(1 * time.Hour) {
		heartbeat()
	}
}

func heartbeat() {
	system, err := models.GetSystem()

	if err != nil {
		log.Error(err)
		continue
	}

	apps, err := models.ListApps()

	if err != nil {
		log.Error(err)
		continue
	}

	helpers.TrackEvent("kernel-heartbeat", map[string]interface{}{
		"app_count":      len(apps),
		"instance_count": system.Count,
		"instance_type":  system.Type,
		"version":        system.Version,
	})
}
