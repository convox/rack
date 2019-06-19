package local

import (
	"context"
	"time"
)

func (p *Provider) watchForProcessCompletion(ctx context.Context, app, pid string, cancel func()) {
	defer cancel()

	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			if ps, err := p.ProcessGet(app, pid); err != nil || (ps != nil && ps.Status == "complete") {
				time.Sleep(2 * time.Second)
				cancel()
				return
			}
		}
	}
}
