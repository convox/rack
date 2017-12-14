package local

import (
	"fmt"

	"github.com/convox/rack/structs"
)

func (p *Provider) EventSend(*structs.Event, error) error {
	return fmt.Errorf("unimplemented")
}
