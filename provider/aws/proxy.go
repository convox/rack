package aws

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/structs"
)

func (p *Provider) Proxy(host string, port int, rw io.ReadWriter, opts structs.ProxyOptions) error {
	cn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 3*time.Second)
	if err != nil {
		return err
	}

	if cb(opts.TLS, false) {
		cn = tls.Client(cn, &tls.Config{})
	}

	if err := helpers.Pipe(cn, rw); err != nil {
		return err
	}

	return nil
}
