package base

import (
	"fmt"
	"io"

	"github.com/convox/rack/pkg/structs"
)

func (p *Provider) BuildCreate(app, url string, opts structs.BuildCreateOptions) (*structs.Build, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) BuildExport(app, id string, w io.Writer) error {
	return fmt.Errorf("unimplemented")
}

func (p *Provider) BuildGet(app, id string) (*structs.Build, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) BuildImport(app string, r io.Reader) (*structs.Build, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) BuildLogs(app, id string, opts structs.LogsOptions) (io.ReadCloser, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) BuildList(app string, opts structs.BuildListOptions) (structs.Builds, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) BuildUpdate(app, id string, opts structs.BuildUpdateOptions) (*structs.Build, error) {
	return nil, fmt.Errorf("unimplemented")
}
