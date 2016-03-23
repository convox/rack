package client

import (
	"fmt"
	"io"
)

func (c *Client) Proxy(host string, port int, rw io.ReadWriteCloser) error {
	return c.Stream(fmt.Sprintf("/proxy/%s/%d", host, port), nil, rw, rw)
}
