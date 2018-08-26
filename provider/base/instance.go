package base

import (
	"fmt"
	"io"

	"github.com/convox/rack/pkg/structs"
)

func (p *Provider) InstanceKeyroll() error {
	return fmt.Errorf("unimplemented")
}

func (p *Provider) InstanceList() (structs.Instances, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) InstanceShell(id string, rw io.ReadWriter, opts structs.InstanceShellOptions) (int, error) {
	return 0, fmt.Errorf("unimplemented")
}

func (p *Provider) InstanceTerminate(id string) error {
	return fmt.Errorf("unimplemented")
}
