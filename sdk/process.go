package sdk

import (
	"fmt"

	"github.com/convox/rack/structs"
)

func (c *Client) ProcessList(app string, opts structs.ProcessListOptions) (structs.Processes, error) {
	ro, err := marshalOptions(opts)
	if err != nil {
		return nil, err
	}

	var pss structs.Processes

	if err := c.Get(fmt.Sprintf("/apps/%s/processes", app), ro, &pss); err != nil {
		return nil, err
	}

	return pss, nil
}

func (c *Client) ProcessStop(app, pid string) error {
	if err := c.Delete(fmt.Sprintf("/apps/%s/processes/%s", app, pid), RequestOptions{}, nil); err != nil {
		return err
	}

	return nil
}
