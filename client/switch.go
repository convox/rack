package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

func (c *Client) Switch(rackName string) (success map[string]string, err error) {
	form := url.Values{}
	form.Set("rack-name", rackName)
	body := strings.NewReader(form.Encode())
	url := fmt.Sprintf("https://%s/switch", c.Host)

	req, err := http.NewRequest("POST", url, body)

	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("convox", string(c.Password))

	resp, err := c.client().Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if err := responseError(resp); err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &success)

	if err != nil {
		return nil, err
	}

	return success, nil
}
