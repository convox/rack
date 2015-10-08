package client

import "fmt"

func (c *Client) CreateLink(app, name string) (*Service, error) {
	params := Params{
		"app": app,
	}

	var service Service

	err := c.Post(fmt.Sprintf("/services/%s/links", name), params, &service)

	if err != nil {
		return nil, err
	}

	return &service, nil
}

func (c *Client) DeleteLink(app, name string) (*Service, error) {
	var service Service

	err := c.Delete(fmt.Sprintf("/services/%s/links/%s", name, app), &service)

	if err != nil {
		return nil, err
	}

	return &service, nil
}
