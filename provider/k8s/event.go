package k8s

import (
	"fmt"

	"github.com/convox/rack/structs"
)

func (p *Provider) EventSend(action string, opts structs.EventSendOptions) error {
	return fmt.Errorf("unimplemented")
}
