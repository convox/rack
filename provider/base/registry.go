package aws

import (
	"fmt"

	"github.com/convox/rack/structs"
)

func (p *Provider) RegistryAdd(server, username, password string) (*structs.Registry, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) RegistryList() (structs.Registries, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) RegistryRemove(server string) error {
	return fmt.Errorf("unimplemented")
}
