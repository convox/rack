package client

import (
	"fmt"
	"net/http"
)

func (c *Client) Auth() error {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://%s/auth", c.Host), nil)

	if err != nil {
		return err
	}

	req.SetBasicAuth("convox", string(c.Password))

	resp, err := c.client().Do(req)

	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("invalid login\nHave you created an account at https://convox.com/signup?")
	}

	resp.Body.Close()
	return nil
}
