package local

import "fmt"

func (p *Provider) DomainExternal(app, service string) string {
	return fmt.Sprintf("%s.%s.%s", service, app, p.DomainExternalBase())
}

func (p *Provider) DomainExternalBase() string {
	return p.Rack
}

func (p *Provider) DomainInternal(app, service string) string {
	return fmt.Sprintf("%s.%s.%s.convox", service, app, p.DomainInternalBase())
}

func (p *Provider) DomainInternalBase() string {
	return fmt.Sprintf("%s.convox", p.Rack)
}
