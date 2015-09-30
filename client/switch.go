package client

func (c *Client) Switch(rackName string) (success map[string]string, err error) {
	params := map[string]string{
		"rack-name": rackName,
	}

	err = c.Post("/switch", params, &success)
	return
}
