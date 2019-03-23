package k8s

import (
	"fmt"
	"time"
)

func (p *Provider) systemLog(group, name string, ts time.Time, message string) error {
	return p.Engine.Log(group, fmt.Sprintf("system/k8s/%s", name), ts, message)
}
