package sdk

import (
	"fmt"

	"github.com/convox/rack/structs"
)

func (c *Client) ServiceList(app string) (structs.Services, error) {
	var ss structs.Services

	if err := c.Get(fmt.Sprintf("/apps/%s/services", app), RequestOptions{}, &ss); err != nil {
		return nil, err
	}

	return ss, nil
}
