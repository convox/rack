package base

import (
	"fmt"

	"github.com/convox/rack/pkg/structs"
)

func (p *Provider) ServiceList(app string) (structs.Services, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) ServiceMetrics(app, name string, opts structs.MetricsOptions) (structs.Metrics, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) ServiceRestart(app, name string) error {
	return fmt.Errorf("unimplemented")
}

func (p *Provider) ServiceUpdate(app, name string, opts structs.ServiceUpdateOptions) error {
	return fmt.Errorf("unimplemented")
}
