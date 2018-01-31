package client

import (
	"fmt"
	"io"
	"time"
)

type App struct {
	Generation string `json:"generation"`
	Name       string `json:"name"`
	Release    string `json:"release"`
	Status     string `json:"status"`
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

func (c *Client) CreateApp(name, generation string) (*App, error) {
	params := Params{
		"generation": generation,
		"name":       name,
	}

	var app App

	err := c.Post("/apps", params, &app)

	if err != nil {
		return nil, err
	}

	return &app, nil
}

func (c *Client) CancelApp(name string) error {
	err := c.Post(fmt.Sprintf("/apps/%s/cancel", name), nil, nil)
	if err != nil {
		return err
	}

	return nil
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

func (c *Client) StreamAppLogs(app, filter string, follow bool, since time.Duration, output io.WriteCloser) error {
	return c.Stream(fmt.Sprintf("/apps/%s/logs", app), map[string]string{
		"Filter": filter,
		"Follow": fmt.Sprintf("%t", follow),
		"Since":  since.String(),
	}, nil, output)
}
