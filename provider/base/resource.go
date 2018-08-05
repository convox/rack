package aws

import (
	"fmt"

	"github.com/convox/rack/structs"
)

func (p *Provider) ResourceCreate(kind string, opts structs.ResourceCreateOptions) (*structs.Resource, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) ResourceDelete(name string) error {
	return fmt.Errorf("unimplemented")
}

func (p *Provider) ResourceGet(name string) (*structs.Resource, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) ResourceLink(name, app string) (*structs.Resource, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) ResourceList() (structs.Resources, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) ResourceTypes() (structs.ResourceTypes, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) ResourceUnlink(name, app string) (*structs.Resource, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) ResourceUpdate(name string, opts structs.ResourceUpdateOptions) (*structs.Resource, error) {
	return nil, fmt.Errorf("unimplemented")
}
