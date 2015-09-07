package client

import "fmt"

type App struct {
	Balancer string `json:"balancer"`
	Name     string `json:"name"`
	Release  string `json:"release"`
	Status   string `json:"status"`
}

type Apps []App

func (c *Client) GetApps() (Apps, error) {
	var apps Apps

	err := c.Get("/apps", &apps)

	if err != nil {
		return nil, err
	}

	return apps, nil
}

func (c *Client) CreateApp(name string) (*App, error) {
	params := Params{
		"name": name,
	}

	var app App

	err := c.Post("/apps", params, &app)

	if err != nil {
		return nil, err
	}

	return &app, nil
}

func (c *Client) GetApp(name string) (*App, error) {
	var app App

	err := c.Get(fmt.Sprintf("/apps/%s", name), &app)

	if err != nil {
		return nil, err
	}

	return &app, nil
}

func (c *Client) DeleteApp(name string) (*App, error) {
	var app App

	err := c.Delete(fmt.Sprintf("/apps/%s", name), &app)

	if err != nil {
		return nil, err
	}

	return &app, nil
}
