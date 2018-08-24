package aws

import (
	"time"

	"github.com/cloudflare/cfssl/log"
	"github.com/convox/logger"
	"github.com/convox/rack/pkg/helpers"
)

func (p *Provider) workerHeartbeat() {
	log := logger.New("ns=workers.heartbeat")

	defer recoverWith(func(err error) {
		helpers.Error(log, err)
	})

	p.heartbeat()

	for range time.Tick(1 * time.Hour) {
		p.heartbeat()
	}
}

func (p *Provider) heartbeat() {
	system, err := p.SystemGet()
	if err != nil {
		log.Error(err)
		return
	}

	apps, err := p.AppList()
	if err != nil {
		log.Error(err)
		return
	}

	helpers.TrackEvent("kernel-heartbeat", map[string]interface{}{
		"app_count":      len(apps),
		"instance_count": system.Count,
		"instance_type":  system.Type,
		"region":         p.Region,
		"version":        system.Version,
	})
}
