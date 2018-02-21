package router

import (
	"fmt"
)

func (d *DNS) setupResolver(domain string) error {
	data := []byte(fmt.Sprintf("nameserver %s\nport 53\n", d.router.ip))

	if err := writeFile(fmt.Sprintf("/etc/resolver/%s", domain), data); err != nil {
		return err
	}

	return nil
}
