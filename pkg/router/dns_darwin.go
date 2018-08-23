package router

import (
	"fmt"
	"net"
)

func (d *DNS) setupResolver(domain string, ip net.IP) error {
	data := []byte(fmt.Sprintf("nameserver %s\nport 53\n", ip))

	if err := writeFile(fmt.Sprintf("/etc/resolver/%s", domain), data); err != nil {
		return err
	}

	return nil
}
