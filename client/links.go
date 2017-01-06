package client

import "fmt"

// CreateLink creates a link between an app and a resource.
func (c *Client) CreateLink(app, name string) (*Resource, error) {
	params := Params{
		"app": app,
	}

	var resource Resource

	err := c.Post(fmt.Sprintf("/resources/%s/links", name), params, &resource)

	if err != nil {
		return nil, err
	}

	return &resource, nil
}

// DeleteLink deletes a link between an app and a resource.
func (c *Client) DeleteLink(app, name string) (*Resource, error) {
	var resource Resource

	err := c.Delete(fmt.Sprintf("/resources/%s/links/%s", name, app), &resource)

	if err != nil {
		return nil, err
	}

	return &resource, nil
}
