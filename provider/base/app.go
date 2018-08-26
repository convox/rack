package base

import (
	"fmt"
	"io"

	"github.com/convox/rack/pkg/structs"
)

func (p *Provider) AppCancel(name string) error {
	return fmt.Errorf("unimplemented")
}

func (p *Provider) AppCreate(name string, opts structs.AppCreateOptions) (*structs.App, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) AppGet(name string) (*structs.App, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) AppDelete(name string) error {
	return fmt.Errorf("unimplemented")
}

func (p *Provider) AppList() (structs.Apps, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) AppLogs(name string, opts structs.LogsOptions) (io.Reader, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) AppUpdate(name string, opts structs.AppUpdateOptions) error {
	return fmt.Errorf("unimplemented")
}
