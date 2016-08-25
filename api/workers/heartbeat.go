package workers

import (
	"os"
	"time"

	"github.com/cloudflare/cfssl/log"
	"github.com/convox/logger"
	"github.com/convox/rack/api/helpers"
	"github.com/convox/rack/api/models"
)

func StartHeartbeat() {
	log := logger.New("ns=heartbeat")

	defer recoverWith(func(err error) {
		helpers.Error(log, err)
	})

	heartbeat()

	for range time.Tick(1 * time.Hour) {
		heartbeat()
	}
}

func heartbeat() {
	system, err := models.Provider().SystemGet()

	if err != nil {
		log.Error(err)
		return
	}

	apps, err := models.ListApps()

	if err != nil {
		log.Error(err)
		return
	}

	helpers.TrackEvent("kernel-heartbeat", map[string]interface{}{
		"app_count":      len(apps),
		"instance_count": system.Count,
		"instance_type":  system.Type,
		"region":         os.Getenv("AWS_REGION"),
		"version":        system.Version,
	})
}
