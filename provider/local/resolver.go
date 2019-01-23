package local

import (
	"fmt"
	"net"
)

func (p *Provider) Resolver() (string, error) {
	ips, err := net.LookupIP("resolver-internal.convox-system.svc.cluster.local")
	if err != nil {
		return "", err
	}

	if len(ips) == 0 {
		return "", fmt.Errorf("could not look up resolver ip")
	}

	return ips[0].String(), nil
}
