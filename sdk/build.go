package sdk

import (
	"fmt"
	"io"

	"github.com/convox/rack/structs"
)

func (c *Client) BuildCreate(app, method, source string, opts structs.BuildCreateOptions) (*structs.Build, error) {
	ro, err := marshalOptions(opts)
	if err != nil {
		return nil, err
	}

	ro.Params["method"] = method
	ro.Params["url"] = source

	var build structs.Build

	if err := c.Post(fmt.Sprintf("/apps/%s/builds", app), ro, &build); err != nil {
		return nil, err
	}

	return &build, nil
}

func (c *Client) BuildGet(app, id string) (*structs.Build, error) {
	var build structs.Build

	if err := c.Get(fmt.Sprintf("/apps/%s/builds/%s", app, id), RequestOptions{}, &build); err != nil {
		return nil, err
	}

	return &build, nil
}

func (c *Client) BuildLogs(app, id string, opts structs.LogsOptions) (io.ReadCloser, error) {
	return c.Websocket(fmt.Sprintf("/apps/%s/builds/%s/logs", app, id), RequestOptions{})
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
