package client

import "fmt"

type FormationEntry struct {
	Balancer string `json:"balancer"`
	Name     string `json:"name"`
	Count    int    `json:"count"`
	Memory   int    `json:"memory"`
	CPU      int    `json:"cpu"`
	Ports    []int  `json:"ports"`
}

type Formation []FormationEntry

// FormationOptions carries the numeric dimensions that can change for a process type.
// Empty string indicates no change.
type FormationOptions struct {
	Count  string
	CPU    string
	Memory string
}

func (c *Client) ListFormation(app string) (Formation, error) {
	var formation Formation

	err := c.Get(fmt.Sprintf("/apps/%s/formation", app), &formation)
	if err != nil {
		return nil, err
	}

	return formation, nil
}

// SetFormation updates the Count, CPU, or Memory parameters for a process
func (c *Client) SetFormation(app, process string, opts FormationOptions) error {
	var success interface{}

	params := map[string]string{}

	if opts.Count != "" {
		params["count"] = opts.Count
	}

	if opts.CPU != "" {
		params["cpu"] = opts.CPU
	}

	if opts.Memory != "" {
		params["memory"] = opts.Memory
	}

	err := c.Post(fmt.Sprintf("/apps/%s/formation/%s", app, process), params, &success)
	if err != nil {
		return err
	}

	return nil
}
