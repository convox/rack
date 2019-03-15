package sdk

import (
	"encoding/json"
	"io/ioutil"

	"github.com/convox/stdsdk"
)

func (c *Client) Auth() (string, error) {
	res, err := c.GetStream("/auth", stdsdk.RequestOptions{})
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	var auth struct {
		Id string
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	if err := json.Unmarshal(data, &auth); err == nil {
		return auth.Id, nil
	}

	return "", nil
}
