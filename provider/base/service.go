package aws

import (
	"fmt"

	"github.com/convox/rack/structs"
)

func (p *Provider) ServiceList(app string) (structs.Services, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) ServiceUpdate(app, name string, opts structs.ServiceUpdateOptions) error {
	return fmt.Errorf("unimplemented")
}
