package klocal

import "fmt"

func (p *Provider) AppRepository(app string) (string, bool, error) {
	return fmt.Sprintf("%s/%s", p.Rack, app), false, nil
}
