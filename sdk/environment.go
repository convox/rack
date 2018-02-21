package sdk

import (
	"fmt"

	"github.com/convox/rack/structs"
)

func (c *Client) EnvironmentGet(app string) (structs.Environment, error) {
	var env structs.Environment

	if err := c.Get(fmt.Sprintf("/apps/%s/environment", app), RequestOptions{}, &env); err != nil {
		return nil, err
	}

	return env, nil
}
