package local

import (
	"fmt"
	"time"

	"github.com/convox/rack/pkg/helpers"
)

func (p *Provider) Workers() error {
	if p.Test {
		return nil
	}

	go helpers.Tick(10*time.Second, p.workerConverge)
	go helpers.Tick(1*time.Hour, p.workerHeartbeat)

	return nil
}

func (p *Provider) workerConverge() {
	log := p.logger("workerConverge")

	if _, err := p.router.RackGet(p.Rack); err != nil {
		if err := p.routerRegister(); err != nil {
			log.At("register").Error(err)
			return
		}
	}

	if err := p.idle(); err != nil {
		log.At("idle").Error(err)
		return
	}

	apps, err := p.AppList()
	if err != nil {
		log.At("list").Error(err)
		return
	}

	for _, a := range apps {
		if err := p.converge(a.Name); err != nil {
			log.At("converge").Append("app=%s", a.Name).Error(err)
			continue
		}
	}
}

func (p *Provider) workerHeartbeat() {
	as, err := p.AppList()
	if err != nil {
		return
	}

	p.Metrics.Post("heartbeat", map[string]interface{}{
		"id":        p.Id,
		"app_count": len(as),
		"rack_id":   fmt.Sprintf("%s:%s", p.Id, p.Rack),
		"provider":  "local",
		"version":   p.Version,
	})
}
