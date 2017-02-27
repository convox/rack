package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

//Auth is a request that simply checks whether the password is valid
//in the case of console the users id will be returned, a rack will
//return an empty string
func (c *Client) Auth() (string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://%s/auth", c.Host), nil)
	if err != nil {
		return "", err
	}

	req.SetBasicAuth("convox", string(c.Password))

	resp, err := c.client().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("invalid login\nHave you created an account at https://convox.com/signup?")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var id string
	var data map[string]string

	err = json.Unmarshal(body, &data)
	if err == nil {
		//if bad JSON is returned it is probably a legacy rack
		//which used to return the plain text string 'OK'
		id = data["id"]
	}

	return id, nil
}
