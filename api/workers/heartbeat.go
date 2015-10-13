package workers

import (
	"time"

	"github.com/convox/rack/api/Godeps/_workspace/src/github.com/ddollar/logger"
	"github.com/convox/rack/api/helpers"
)

func StartHeartbeat() {
	log := logger.New("ns=heartbeat")
	defer recoverWith(func(err error) {
		helpers.Error(log, err)
	})

	helpers.SendMixpanelEvent("kernel-heartbeat", "")

	for _ = range time.Tick(1 * time.Hour) {
		helpers.SendMixpanelEvent("kernel-heartbeat", "")
	}
}
