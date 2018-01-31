package sdk

import (
	"fmt"

	"github.com/convox/rack/structs"
)

func (c *Client) BuildGet(app, id string) (*structs.Build, error) {
	var build structs.Build

	if err := c.Get(fmt.Sprintf("/apps/%s/builds/%s", app, id), RequestOptions{}, &build); err != nil {
		return nil, err
	}

	return &build, nil
}

func (c *Client) BuildUpdate(app, id string, opts structs.BuildUpdateOptions) (*structs.Build, error) {
	ro, err := marshalOptions(opts)
	if err != nil {
		return nil, err
	}

	var build structs.Build

	if err := c.Put(fmt.Sprintf("/apps/%s/builds/%s", app, id), ro, &build); err != nil {
		return nil, err
	}

	return &build, nil
}
