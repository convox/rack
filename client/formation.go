package client

import (
	"fmt"
	"strconv"
)

type FormationEntry struct {
	Balancer string `json:"balancer"`
	Name     string `json:"name"`
	Count    int    `json:"count"`
	Memory   int    `json:"memory"`
	Ports    []int  `json:"ports"`
}

type Formation []FormationEntry

func (c *Client) ListFormation(app string) (Formation, error) {
	var formation Formation

	err := c.Get(fmt.Sprintf("/apps/%s/formation", app), &formation)

	if err != nil {
		return nil, err
	}

	return formation, nil
}

func (c *Client) SetFormation(app, process string, count, memory int) error {
	var success interface{}

	params := map[string]string{}
	params["count"] = strconv.Itoa(count)
	params["memory"] = strconv.Itoa(memory)

	err := c.Post(fmt.Sprintf("/apps/%s/formation/%s", app, process), params, &success)
	if err != nil {
		return err
	}

	return nil
}
