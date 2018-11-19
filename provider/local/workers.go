package local

import (
	"fmt"
	"time"

	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/metrics"
)

func (p *Provider) Workers() error {
	if err := p.Provider.Workers(); err != nil {
		return err
	}

	go helpers.Tick(1*time.Hour, p.workerHeartbeat)

	return nil
}

func (p *Provider) workerHeartbeat() {
	as, err := p.AppList()
	if err != nil {
		return
	}

	metrics.New("https://metrics.convox.com/metrics/rack").Post("heartbeat", map[string]interface{}{
		"id":        p.ID,
		"app_count": len(as),
		"rack_id":   fmt.Sprintf("%s:%s", p.ID, p.Rack),
		"provider":  "local",
		"version":   p.Version,
	})
}
