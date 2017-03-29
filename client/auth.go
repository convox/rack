package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

//Auth is used to transfer the id of the authenticating user when authing against the console API.
//When authing against a rack the id will be blank as user ids are a console concept
type Auth struct {
	ID string `json:"id"`
}

//Auth is a request that simply checks whether the password is valid
//in the case of console the users id will be returned, a rack will
//return an empty string
func (c *Client) Auth() (*Auth, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://%s/auth", c.Host), nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth("convox", string(c.Password))

	resp, err := c.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("invalid login\nHave you created an account at https://convox.com/signup?")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	auth := &Auth{}

	// legacy racks return a body of "OK\n"
	// legacy consoles return an empty body
	if string(body) == "OK\n" || len(body) == 0 {
		return auth, nil
	}

	err = json.Unmarshal(body, &auth)
	if err != nil {
		return auth, err
	}

	return auth, nil
}
