package client

import (
	"fmt"
	"io"
	"time"
)

type Rack struct {
	Organization struct {
		Name string `json:"name"`
	}
	Name   string `json:"name"`
	Status string `json:"status"`
}

func (c *Client) Racks() (racks []Rack, err error) {
	err = c.Get("/racks", &racks)
	return racks, err
}

// StreamRackLogs streams the logs for a Rack
func (c *Client) StreamRackLogs(filter string, follow bool, since time.Duration, output io.WriteCloser) error {
	return c.Stream("/system/logs", map[string]string{
		"Filter": filter,
		"Follow": fmt.Sprintf("%t", follow),
		"Since":  since.String(),
	}, nil, output)
}
