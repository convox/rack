package local

import "fmt"

func (p *Provider) ServiceHost(app, service string) string {
	return fmt.Sprintf("%s.%s", service, app)
}
