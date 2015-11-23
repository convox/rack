package client

import "fmt"

type InstanceResource struct {
	Total int `json:"total"`
	Free  int `json:"free"`
	Used  int `json:"used"`
}

func (ir InstanceResource) PercentUsed() float64 {
	return float64(ir.Used) / float64(ir.Total)
}

type Instance struct {
	Agent   bool   `json:"agent"`
	Id      string `json:"id"`
	Running int    `json:"running"`
	Pending int    `json:"pending"`
	Status  string `json:"status"`
	Cpu     InstanceResource
	Memory  InstanceResource
}

func (c *Client) GetInstances() ([]*Instance, error) {
	var instances []*Instance

	err := c.Get("/system/instances", &instances)

	if err != nil {
		return nil, err
	}

	return instances, nil
}

func (c *Client) TerminateInstance(id string) error {

	err := c.Delete(fmt.Sprintf("/system/instance/%s", id), nil)

	if err != nil {
		return err
	}

	return nil
}
