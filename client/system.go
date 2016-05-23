package client

import "strconv"

type System struct {
	Count   int    `json:"count"`
	Name    string `json:"name"`
	Region  string `json:"region"`
	Status  string `json:"status"`
	Type    string `json:"type"`
	Version string `json:"version"`
}

type SystemCapacity struct {
	ClusterMemory  int64 `json:"cluster-memory"`
	InstanceMemory int64 `json:"instance-memory"`
	ProcessCount   int64 `json:"process-count"`
	ProcessMemory  int64 `json:"process-memory"`
	ProcessWidth   int64 `json:"process-width"`
}

func (c *Client) GetSystem() (*System, error) {
	var system System

	err := c.Get("/system", &system)

	if err != nil {
		return nil, err
	}

	if system.Version == "" {
		var sys struct {
			Outputs    map[string]string
			Parameters map[string]string
		}

		err = c.Get("/system", &sys)

		if err != nil {
			return nil, err
		}

		system.Count, _ = strconv.Atoi(sys.Parameters["InstanceCount"])
		system.Type = sys.Parameters["InstanceType"]
		system.Version = sys.Parameters["Version"]
	}

	return &system, nil
}

func (c *Client) GetSystemCapacity() (*SystemCapacity, error) {
	var capacity SystemCapacity

	err := c.Get("/system/capacity", &capacity)

	if err != nil {
		return nil, err
	}

	return &capacity, nil
}

func (c *Client) GetSystemReleases() (Releases, error) {
	var releases Releases

	err := c.Get("/system/releases", &releases)

	if err != nil {
		return nil, err
	}

	return releases, nil
}

func (c *Client) UpdateSystem(version string) (*System, error) {
	var system System

	err := c.Get("/system", &system)

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
	err := c.Post("/system", map[string]string{"version": version}, nil)

	if err != nil {
		return nil, err
	}

	return c.GetSystem()
}

func (c *Client) ScaleSystem(count int, typ string) (*System, error) {
	var system System

	params := Params{}

	params["count"] = strconv.Itoa(count)
	params["type"] = typ

	err := c.Put("/system", params, &system)

	if err != nil {
		return nil, err
	}

	return &system, nil
}
