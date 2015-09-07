package client

import "fmt"

type Process struct {
	Id      string `json:"id"`
	App     string `json:"app"`
	Command string `json:"command"`
	Image   string `json:"image"`
	Name    string `json:"name"`
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
