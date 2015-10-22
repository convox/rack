package client

import (
	"fmt"
	"time"
)

type SSL struct {
	Id         string    `json:"id"`
	Arn        string    `json:"arn"`
	Expiration time.Time `json:"expiration"`
	Name       string    `json:"name"`
	Port       int       `json:"port"`
	Process    string    `json:"process"`
}

type SSLs []SSL

func (c *Client) CreateSSL(app, process, port, body, key string) (*SSL, error) {
	params := Params{
		"body":    body,
		"key":     key,
		"port":    port,
		"process": process,
	}

	var ssl SSL

	err := c.Post(fmt.Sprintf("/apps/%s/ssl", app), params, &ssl)

	if err != nil {
		return nil, err
	}

	return &ssl, nil
}

func (c *Client) DeleteSSL(app, process, port string) (*SSL, error) {
	var ssl SSL

	err := c.Delete(fmt.Sprintf("/apps/%s/ssl/%s/%s", app, process, port), &ssl)

	if err != nil {
		return nil, err
	}

	return &ssl, nil
}

func (c *Client) ListSSL(app string) (*SSLs, error) {
	var ssls SSLs

	err := c.Get(fmt.Sprintf("/apps/%s/ssl", app), &ssls)

	if err != nil {
		return nil, err
	}

	return &ssls, nil
}
