package base

import (
	"fmt"

	"github.com/convox/rack/pkg/structs"
)

func (p *Provider) ReleaseCreate(app string, opts structs.ReleaseCreateOptions) (*structs.Release, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) ReleaseGet(app, id string) (*structs.Release, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) ReleaseList(app string, opts structs.ReleaseListOptions) (structs.Releases, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) ReleasePromote(app, id string, opts structs.ReleasePromoteOptions) error {
	return fmt.Errorf("unimplemented")
}
