package client

import "strconv"

type System struct {
	Count   int    `json:"count"`
	Name    string `json:"name"`
	Status  string `json:"status"`
	Type    string `json:"type"`
	Version string `json:"version"`
}

func (c *Client) GetSystem() (*System, error) {
	var system System

	err := c.Get("/system", &system)

	if err != nil {
		return nil, err
	}

	return &system, nil
}

func (c *Client) UpdateSystem(version string) (*System, error) {
	var system System

	params := Params{
		"version": version,
	}

	err := c.Put("/system", params, &system)

	if err != nil {
		return nil, err
	}

	return &system, nil
}

func (c *Client) ScaleSystem(count int, typ string) (*System, error) {
	var system System

	params := Params{}

	if count > 0 {
		params["count"] = strconv.Itoa(count)
	}

	if typ != "" {
		params["type"] = typ
	}

	err := c.Put("/system", params, &system)

	if err != nil {
		return nil, err
	}

	return &system, nil
}
