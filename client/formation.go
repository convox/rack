package client

import "fmt"

type FormationEntry struct {
	Name   string `json:"name"`
	Count  int    `json:"count"`
	Memory int    `json:"memory"`
	Ports  []int  `json:"ports"`
}

type Formation []FormationEntry

func (c *Client) GetFormation(app string) (Formation, error) {
	var formation Formation

	err := c.Get(fmt.Sprintf("/apps/%s/formation", app), &formation)

	if err != nil {
		return nil, err
	}

	return formation, nil
}
