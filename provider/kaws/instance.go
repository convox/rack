package kaws

import (
	"fmt"

	"github.com/convox/rack/pkg/structs"
)

func (p *Provider) InstanceList() (structs.Instances, error) {
	is, err := p.Provider.InstanceList()
	if err != nil {
		return nil, err
	}

	fmt.Println("kaws additions")

	return is, nil
}
