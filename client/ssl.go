package client

import (
	"fmt"
	"time"
)

type SSL struct {
	Domain     string    `json:"domain"`
	Expiration time.Time `json:"expiration"`
	Port       int       `json:"port"`
	Process    string    `json:"process"`
	Secure     bool      `json:"secure"`
}

type SSLs []SSL

func (c *Client) ListSSL(app string) (*SSLs, error) {
	var ssls SSLs

	err := c.Get(fmt.Sprintf("/apps/%s/ssl", app), &ssls)

	if err != nil {
		return nil, err
	}

	return &ssls, nil
}

func (c *Client) UpdateSSL(app, process, port, arn, body, key, chain string) (*SSL, error) {
	params := Params{
		"arn":   arn,
		"body":  body,
		"chain": chain,
		"key":   key,
	}

	var ssl SSL

	err := c.Put(fmt.Sprintf("/apps/%s/ssl/%s/%s", app, process, port), params, &ssl)

	if err != nil {
		return nil, err
	}

	return &ssl, nil
}
