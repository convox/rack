package workers

import (
	"os"
	"time"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/cloudflare/cfssl/log"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/ddollar/logger"
	"github.com/convox/rack/api/helpers"
	"github.com/convox/rack/api/models"
	"github.com/convox/rack/api/provider"
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
	system, err := provider.SystemGet()

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
