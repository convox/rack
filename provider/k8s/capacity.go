package k8s

import (
	"fmt"

	"github.com/convox/rack/structs"
)

func (p *Provider) CapacityGet() (*structs.Capacity, error) {
	return nil, fmt.Errorf("unimplemented")
}
