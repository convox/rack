package aws

import (
	"fmt"
	"io"
	"net"
	"time"

	"github.com/convox/rack/helpers"
)

func (p *AWSProvider) Proxy(host string, port int, rw io.ReadWriter) error {
	cn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 3*time.Second)
	if err != nil {
		return err
	}

	if err := helpers.Pipe(cn, rw); err != nil {
		return err
	}

	return nil
}
