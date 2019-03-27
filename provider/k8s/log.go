package k8s

import (
	"fmt"
	"time"
)

func (p *Provider) systemLog(app, name string, ts time.Time, message string) error {
	return p.Engine.Log(app, fmt.Sprintf("system/k8s/%s", name), ts, message)
}
