package client

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/convox/cli/stdcli"
)

type Client struct {
	Host     string
	Password string
}

type Params map[string]string

func New(host, password string) *Client {
	return &Client{
		Host:     host,
		Password: password,
	}
}

func (c *Client) client() *http.Client {
	client := &http.Client{}

	client.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	return client
}

func (c *Client) request(method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, fmt.Sprintf("https://%s%s", c.Host, path), body)

	if err != nil {
		stdcli.Error(err)
		return nil, err
	}

	req.SetBasicAuth("convox", string(c.Password))
	req.Header.Add("Content-Type", "application/json")

	return req, nil
}

func (c *Client) Get(path string, out interface{}) error {
	req, err := c.request("GET", path, nil)

	if err != nil {
		return nil
	}

	res, err := c.client().Do(req)

	if err != nil {
		return err
	}

	defer res.Body.Close()

	if err := responseError(res); err != nil {
		return err
	}

	data, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return err
	}

	err = json.Unmarshal(data, out)

	if err != nil {
		return err
	}

	return nil
}

func (c *Client) Post(path string, params Params, out interface{}) error {
	form := url.Values{}

	for k, v := range params {
		form.Set(k, v)
	}

	req, err := c.request("POST", path, strings.NewReader(form.Encode()))

	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := c.client().Do(req)

	if err != nil {
		return err
	}

	defer res.Body.Close()

	if err := responseError(res); err != nil {
		return err
	}

	data, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return err
	}

	err = json.Unmarshal(data, out)

	if err != nil {
		return err
	}

	return nil
}

func (c *Client) Delete(path string, out interface{}) error {
	req, err := c.request("DELETE", path, nil)

	if err != nil {
		return nil
	}

	res, err := c.client().Do(req)

	if err != nil {
		return err
	}

	defer res.Body.Close()

	if err := responseError(res); err != nil {
		return err
	}

	data, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return err
	}

	err = json.Unmarshal(data, out)

	if err != nil {
		return err
	}

	return nil
}

func (c *Client) url(path string) string {
	return fmt.Sprintf("https://%s%s", c.Host, path)
}
