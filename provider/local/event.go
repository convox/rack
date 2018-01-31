package local

import (
	"fmt"

	"github.com/convox/rack/structs"
)

func (p *Provider) EventSend(action string, opts structs.EventSendOptions) error {
	fmt.Println("EventSend")
	fmt.Printf("action = %+v\n", action)
	fmt.Printf("opts = %+v\n", opts)

	return nil
}
