package client

import (
	"fmt"
	"io"
)

type Environment map[string]string

func (c *Client) GetEnvironment(app string) (Environment, error) {
	var env Environment

	err := c.Get(fmt.Sprintf("/apps/%s/environment", app), &env)

	if err != nil {
		return nil, err
	}

	return env, nil
}

func (c *Client) SetEnvironment(app string, body io.Reader) (Environment, string, error) {
	var env Environment

	res, err := c.PostBodyResponse(fmt.Sprintf("/apps/%s/environment", app), body, &env)

	if err != nil {
		return nil, "", err
	}

	return env, res.Header.Get("Release-Id"), nil
}

func (c *Client) DeleteEnvironment(app, key string) (Environment, string, error) {
	var env Environment

	res, err := c.DeleteResponse(fmt.Sprintf("/apps/%s/environment/%s", app, key), &env)

	if err != nil {
		return nil, "", err
	}

	return env, res.Header.Get("Release-Id"), nil
}
