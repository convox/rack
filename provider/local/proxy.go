package local

import (
	"fmt"
	"io"
)

func (p *Provider) Proxy(host string, port int, rw io.ReadWriter) error {
	return fmt.Errorf("unimplemented")
}
