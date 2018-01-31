package sdk

import (
	"fmt"

	"github.com/convox/rack/structs"
)

func (c *Client) EventSend(action string, opts structs.EventSendOptions) error {
	ro, err := marshalOptions(opts)
	if err != nil {
		return err
	}

	if err := c.Post(fmt.Sprintf("/events/%s", action), ro, nil); err != nil {
		return err
	}

	return nil
}
