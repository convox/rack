package client

import (
	"errors"
	"fmt"
	"io"
)

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

func (c *Client) SSHInstance(id, cmd string, in io.Reader, out io.WriteCloser) (int, error) {
	r, w := io.Pipe()

	defer r.Close()
	defer w.Close()

	ch := make(chan int)

	go copyWithExit(out, r, ch)

	err := c.Stream(fmt.Sprintf("/instances/%s/ssh", id), map[string]string{"Command": cmd}, in, w)

	if err != nil {
		return 0, err
	}

	code := <-ch

	return code, nil
}

func (c *Client) TerminateInstance(id string) error {
	var response map[string]interface{}
	err := c.Delete(fmt.Sprintf("/instances/%s", id), &response)

	if err != nil {
		return err
	}

	if response["success"] == nil {
		return errors.New(response["error"].(string))
	}

	return nil
}
