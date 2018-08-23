package aws

import (
	"context"

	"github.com/convox/rack/pkg/structs"
)

func (p *Provider) WithContext(ctx context.Context) structs.Provider {
	cp := *p
	cp.ctx = ctx
	return &cp
}
