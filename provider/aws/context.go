package aws

import (
	"context"

	"github.com/convox/rack/structs"
)

func (p *AWSProvider) WithContext(ctx context.Context) structs.Provider {
	cp := *p
	cp.ctx = ctx
	return &cp
}
