package client

import (
	"fmt"
	"io"
	"time"
)

type Build struct {
	Id       string `json:"id"`
	App      string `json:"app"`
	Logs     string `json:"logs"`
	Manifest string `json:"manifest"`
	Release  string `json:"release"`
	Status   string `json:"status"`

	Started time.Time `json:"started"`
	Ended   time.Time `json:"ended"`
}

type Builds []Build

func (c *Client) GetBuilds(app string) (Builds, error) {
	var builds Builds

	err := c.Get(fmt.Sprintf("/apps/%s/builds", app), &builds)

	if err != nil {
		return nil, err
	}

	return builds, nil
}

func (c *Client) CreateBuild(app string, source []byte) (*Build, error) {
	var build Build

	err := c.PostMultipart(fmt.Sprintf("/apps/%s/builds", app), source, &build)

	if err != nil {
		return nil, err
	}

	return &build, nil
}

func (c *Client) GetBuild(app, id string) (*Build, error) {
	var build Build

	err := c.Get(fmt.Sprintf("/apps/%s/builds/%s", app, id), &build)

	if err != nil {
		return nil, err
	}

	return &build, nil
}

func (c *Client) StreamBuildLogs(app, id string, output io.Writer) error {
	return c.Stream(fmt.Sprintf("/apps/%s/builds/%s/logs", app, id), nil, nil, output)
}
