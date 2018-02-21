package sdk

import (
	"fmt"

	"github.com/convox/rack/structs"
)

func (c *Client) ReleaseCreate(app string, opts structs.ReleaseCreateOptions) (*structs.Release, error) {
	ro, err := marshalOptions(opts)
	if err != nil {
		return nil, err
	}

	var release structs.Release

	if err := c.Post(fmt.Sprintf("/apps/%s/releases", app), ro, &release); err != nil {
		return nil, err
	}

	return &release, nil
}

func (c *Client) ReleaseGet(app, id string) (*structs.Release, error) {
	var release structs.Release

	if err := c.Get(fmt.Sprintf("/apps/%s/releases/%s", app, id), RequestOptions{}, &release); err != nil {
		return nil, err
	}

	return &release, nil
}

func (c *Client) ReleasePromote(app, id string) error {
	if err := c.Post(fmt.Sprintf("/apps/%s/releases/%s/promote", app, id), RequestOptions{}, nil); err != nil {
		return err
	}

	return nil
}
