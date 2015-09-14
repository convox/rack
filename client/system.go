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

	check := c.skipVersionCheck
	c.skipVersionCheck = true

	err := c.Get("/system", &system)

	c.skipVersionCheck = check

	if err != nil {
		return nil, err
	}

	if system.Version == "" {
		var sys struct {
			Outputs    map[string]string
			Parameters map[string]string
		}

		check := c.skipVersionCheck
		c.skipVersionCheck = true

		err = c.Get("/system", &sys)

		c.skipVersionCheck = check

		if err != nil {
			return nil, err
		}

		system.Count, _ = strconv.Atoi(sys.Parameters["InstanceCount"])
		system.Type = sys.Parameters["InstanceType"]
		system.Version = sys.Parameters["Version"]
	}

	return &system, nil
}

func (c *Client) UpdateSystem(version string) (*System, error) {
	var system System

	check := c.skipVersionCheck
	c.skipVersionCheck = true

	err := c.Get("/system", &system)

	c.skipVersionCheck = check

	if err != nil {
		return nil, err
	}

	if system.Version == "" {
		return c.UpdateSystemOriginal(version)
	}

	params := Params{
		"version": version,
	}

	err = c.Put("/system", params, &system)

	if err != nil {
		return nil, err
	}

	return &system, nil
}

func (c *Client) UpdateSystemOriginal(version string) (*System, error) {
	c.WithoutVersionCheck(func(c *Client) {
		c.Post("/system", map[string]string{"version": version}, nil)
	})

	return c.GetSystem()
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
