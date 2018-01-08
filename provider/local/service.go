package local

import (
	"fmt"

	"github.com/convox/rack/helpers"
	"github.com/convox/rack/structs"
	"github.com/pkg/errors"
)

func (p *Provider) ServiceList(app string) (structs.Services, error) {
	log := p.logger("ServiceList").Append("app=%q", app)

	m, _, err := helpers.AppManifest(p, app)
	if err != nil {
		return nil, errors.WithStack(log.Error(err))
	}

	ss := structs.Services{}

	for _, s := range m.Services {
		domain := ""

		if s.Port.Port > 0 {
			domain = fmt.Sprintf("%s.%s.%s", s.Name, app, p.Name)
		}

		ss = append(ss, structs.Service{
			Name:   s.Name,
			Domain: domain,
		})
	}

	return ss, log.Success()
}

func (p *Provider) ServiceUpdate(app, name string, opts structs.ServiceUpdateOptions) error {
	return fmt.Errorf("unimplemented")
}
