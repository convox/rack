package k8s

import (
	"fmt"

	"github.com/convox/rack/pkg/structs"
)

func (p *Provider) CapacityGet() (*structs.Capacity, error) {
	return nil, fmt.Errorf("unimplemented")
}
