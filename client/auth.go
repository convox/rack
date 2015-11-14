package client

// calls rack Auth endpoint
// returning error or success message
func (outer *Client) Auth() (string, error) {
	var message string
	var err error

	outer.WithoutVersionCheck(func(c *Client) {
		reply := make(map[string]string)
		err = c.Get("/auth", &reply)
		message = reply["message"]
	})

	if err != nil {
		return "", err
	}

	return message, nil
}
