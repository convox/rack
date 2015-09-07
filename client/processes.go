package client

import "fmt"

type Process struct {
	App     string `json:"app"`
	Command string `json:"command"`
	Count   int    `json:"count"`
	Image   string `json:"image"`
	Memory  int    `json:"memory"`
	Name    string `json:"name"`
	Ports   []int  `json:"ports"`
}

type Processes []Process

func (c *Client) GetProcesses(app string) (Processes, error) {
	var processes Processes

	err := c.Get(fmt.Sprintf("/apps/%s/processes", app), &processes)

	if err != nil {
		return nil, err
	}

	return processes, nil
}
