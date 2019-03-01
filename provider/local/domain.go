package local

import "fmt"

func (p *Provider) DomainBase() string {
	return p.Rack
}

func (p *Provider) DomainExternal(app, service string) string {
	return fmt.Sprintf("%s.%s.%s", service, app, p.DomainBase())
}

func (p *Provider) DomainInternal(app, service string) string {
	return fmt.Sprintf("%s.%s.%s.convox", service, app, p.DomainBase())
}
