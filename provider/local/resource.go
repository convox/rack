package local

import (
	"fmt"

	"github.com/convox/rack/pkg/structs"
)

func (p *Provider) ResourceGet(app, name string) (*structs.Resource, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) ResourceList(app string) (structs.Resources, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) SystemResourceCreate(kind string, opts structs.ResourceCreateOptions) (*structs.Resource, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) SystemResourceDelete(name string) error {
	return fmt.Errorf("unimplemented")
}

func (p *Provider) SystemResourceGet(name string) (*structs.Resource, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) SystemResourceLink(name, app string) (*structs.Resource, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) SystemResourceList() (structs.Resources, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) SystemResourceTypes() (structs.ResourceTypes, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) SystemResourceUnlink(name, app string) (*structs.Resource, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) SystemResourceUpdate(name string, opts structs.ResourceUpdateOptions) (*structs.Resource, error) {
	return nil, fmt.Errorf("unimplemented")
}
