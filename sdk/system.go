package sdk

import "github.com/convox/rack/structs"

func (c *Client) SystemGet() (*structs.System, error) {
	var s structs.System

	if err := c.Get("/system", RequestOptions{}, &s); err != nil {
		return nil, err
	}

	return &s, nil
}
