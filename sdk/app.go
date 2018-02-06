package sdk

import (
	"fmt"
	"io"

	"github.com/convox/rack/structs"
)

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
