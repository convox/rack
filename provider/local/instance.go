package local

import (
	"fmt"
	"io"

	"github.com/convox/rack/structs"
)

func (p *Provider) InstanceKeyroll() error {
	return fmt.Errorf("unimplemented")
}

func (p *Provider) InstanceList() (structs.Instances, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) InstanceShell(id string, rw io.ReadWriter, opts structs.InstanceShellOptions) error {
	return fmt.Errorf("unimplemented")
}

func (p *Provider) InstanceTerminate(id string) error {
	return fmt.Errorf("unimplemented")
}
