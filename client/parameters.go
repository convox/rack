package client

import "fmt"

type Parameters map[string]string

func (c *Client) ListParameters(app string) (Parameters, error) {
	var formation Parameters

	err := c.Get(fmt.Sprintf("/apps/%s/parameters", app), &formation)

	if err != nil {
		return nil, err
	}

	return formation, nil
}

func (c *Client) SetParameters(app string, params map[string]string) error {
	var success interface{}

	err := c.Post(fmt.Sprintf("/apps/%s/parameters", app), params, &success)

	if err != nil {
		return err
	}

	return nil
}
