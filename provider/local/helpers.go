package local

import (
	"context"
	"time"
)

func (p *Provider) watchForProcessTermination(ctx context.Context, app, pid string, cancel func()) {
	defer cancel()

	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			if _, err := p.ProcessGet(app, pid); err != nil {
				cancel()
				return
			}
		}
	}
}
