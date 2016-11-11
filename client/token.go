package client

func (c *Client) RegenerateToken(email, password string) (string, error) {
	params := Params{
		"email":    email,
		"password": password,
	}

	result := map[string]string{}

	err := c.Post("/token", params, &result)
	if err != nil {
		return "", err
	}

	return result["token"], nil
}
