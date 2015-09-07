package client

import (
	"fmt"
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

func (c *Client) CreateBuild(app, repo string) (*Build, error) {
	params := Params{}

	var build Build

	err := c.Post(fmt.Sprintf("/apps/%s/builds", app), params, &build)

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
