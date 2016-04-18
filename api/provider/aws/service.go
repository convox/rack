package aws

import (
	"fmt"

	"github.com/convox/rack/api/structs"
)

func (p *AWSProvider) ServiceCreate(name, kind string, params map[string]string) (*structs.Service, error) {
	return nil, fmt.Errorf("not yet implemented")
}
