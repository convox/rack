package sdk

import "github.com/convox/stdsdk"

func (c *Client) Auth() error {
	return c.Get("/auth", stdsdk.RequestOptions{}, nil)
}
