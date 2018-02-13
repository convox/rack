package sdk

import (
	"fmt"
	"io"

	"github.com/convox/rack/structs"
)

func (c *Client) AppCreate(name string, opts structs.AppCreateOptions) (*structs.App, error) {
	ro, err := marshalOptions(opts)
	if err != nil {
		return nil, err
	}

	ro.Params["name"] = name

	var app structs.App

	if err := c.Post("/apps", ro, &app); err != nil {
		return nil, err
	}

	return &app, nil
}

func (c *Client) AppGet(name string) (*structs.App, error) {
	var app structs.App

	if err := c.Get(fmt.Sprintf("/apps/%s", name), RequestOptions{}, &app); err != nil {
		return nil, err
	}

	return &app, nil
}

func (c *Client) AppLogs(app string, opts structs.LogsOptions) (io.ReadCloser, error) {
	return c.Websocket(fmt.Sprintf("/apps/%s/logs", app), RequestOptions{})
}
