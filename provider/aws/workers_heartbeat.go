package aws

import (
	"time"

	"github.com/convox/logger"
	"github.com/convox/rack/pkg/helpers"
)

func (p *Provider) workerHeartbeat() {
	helpers.Tick(1*time.Hour, p.heartbeat)
}

func (p *Provider) heartbeat() {
	var log = logger.New("ns=workers.heartbeat")

	s, err := p.SystemGet()
	if err != nil {
		log.Error(err)
		return
	}

	as, err := p.AppList()
	if err != nil {
		log.Error(err)
		return
	}

	ms := map[string]interface{}{
		"id":             coalesces(p.ClientId, p.StackId),
		"app_count":      len(as),
		"instance_count": s.Count,
		"instance_type":  s.Type,
		"provider":       "aws",
		"rack_id":        p.StackId,
		"region":         p.Region,
		"version":        s.Version,
	}

	telemetryOn := s.Parameters["Telemetry"] == "true"

	if telemetryOn {
		params := p.RackParamsToSync(s.Version, s.Parameters)
		ms["rack_params"] = params
	}

	if err := p.Metrics.Post("heartbeat", ms); err != nil {
		log.Error(err)
		return
	}
}
