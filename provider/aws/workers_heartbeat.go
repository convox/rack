package aws

import (
	"time"

	"github.com/convox/rack/pkg/helpers"
)

func (p *Provider) workerHeartbeat() {
	helpers.Tick(1*time.Hour, p.heartbeat)
}

func (p *Provider) heartbeat() {
	s, err := p.SystemGet()
	if err != nil {
		return
	}

	as, err := p.AppList()
	if err != nil {
		return
	}

	p.Metrics.Post("heartbeat", map[string]interface{}{
		"id":             coalesces(p.ClientId, p.StackId),
		"app_count":      len(as),
		"instance_count": s.Count,
		"instance_type":  s.Type,
		"provider":       "aws",
		"rack_id":        p.StackId,
		"region":         p.Region,
		"version":        s.Version,
	})
}
