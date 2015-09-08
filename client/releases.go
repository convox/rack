package client

import (
	"fmt"
	"io"
	"time"
)

type Release struct {
	Id       string    `json:"id"`
	App      string    `json:"app"`
	Build    string    `json:"build"`
	Env      string    `json:"env"`
	Manifest string    `json:"manifest"`
	Created  time.Time `json:"created"`
}

type Releases []Release

func (c *Client) GetReleases(app string) (Releases, error) {
	var releases Releases

	err := c.Get(fmt.Sprintf("/apps/%s/releases", app), &releases)

	if err != nil {
		return nil, err
	}

	return releases, nil
}

func (c *Client) GetRelease(app, id string) (*Release, error) {
	var release Release

	err := c.Get(fmt.Sprintf("/apps/%s/releases/%s", app, id), &release)

	if err != nil {
		return nil, err
	}

	return &release, nil
}

func (c *Client) PromoteRelease(app, id string) (*Release, error) {
	var release Release

	err := c.Post(fmt.Sprintf("/apps/%s/releases/%s/promote", app, id), nil, &release)

	if err != nil {
		return nil, err
	}

	return &release, nil
}

func (c *Client) StreamReleaseLogs(app, id string, output io.Writer) error {
	return c.Stream(fmt.Sprintf("/apps/%s/releases/%s/logs", app, id), nil, nil, output)
}
