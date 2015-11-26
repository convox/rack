package client

import "fmt"

type Instance struct {
	Agent     bool    `json:"agent"`
	Cpu       float64 `json:"cpu"`
	Id        string  `json:"id"`
	Ip        string  `json:"ip"`
	Memory    float64 `json:"memory"`
	Processes int     `json:"processes"`
	Status    string  `json:"status"`
}

func (c *Client) GetInstances() ([]*Instance, error) {
	var instances []*Instance

	err := c.Get("/instances", &instances)

	if err != nil {
		return nil, err
	}

	return instances, nil
}

func (c *Client) TerminateInstance(id string) error {

	err := c.Delete(fmt.Sprintf("/instance/%s", id), nil)

	if err != nil {
		return err
	}

	return nil
}
