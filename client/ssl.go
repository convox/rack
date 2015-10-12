package client

import "fmt"

type SSL struct {
	Id   string `json:"id"`
	Port string `json:"port"`
}

type SSLs []SSL

func (c *Client) GetSSLs(app string) (SSLs, error) {
	var ssls SSLs

	err := c.Get(fmt.Sprintf("/apps/%s/ssl", app), &ssls)

	if err != nil {
		return nil, err
	}

	return ssls, nil
}

func (c *Client) CreateSSL(app, port, body, key string) (*SSL, error) {
	params := Params{
		"body": body,
		"key":  key,
		"port": port,
	}

	var ssl SSL

	err := c.Post(fmt.Sprintf("/apps/%s/ssl", app), params, &ssl)

	if err != nil {
		return nil, err
	}

	return &ssl, nil
}

func (c *Client) DeleteSSL(app, port string) (*SSL, error) {
	var ssl SSL

	err := c.Delete(fmt.Sprintf("/apps/%s/ssl/%s", app, port), &ssl)

	if err != nil {
		return nil, err
	}

	return &ssl, nil
}
