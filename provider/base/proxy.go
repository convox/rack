package base

import (
	"fmt"
	"io"

	"github.com/convox/rack/pkg/structs"
)

func (p *Provider) Proxy(host string, port int, rw io.ReadWriter, opts structs.ProxyOptions) error {
	return fmt.Errorf("unimplemented")
}
