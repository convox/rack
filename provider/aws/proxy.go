package aws

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/convox/rack/pkg/structs"
	"github.com/convox/stdsdk"
)

func (p *Provider) Proxy(host string, port int, rw io.ReadWriter, opts structs.ProxyOptions) error {
	cn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 5*time.Second)
	if err != nil {
		return err
	}

	if cb(opts.TLS, false) {
		cn = tls.Client(cn, &tls.Config{})
	}

	if err := stdsdk.CopyStreamToEachOther(cn, rw); err != nil {
		p.log.Errorf("proxy %s", err)
		return err
	}

	return nil
}
