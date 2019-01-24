package local

import (
	"fmt"
	"html/template"

	"github.com/convox/rack/pkg/helpers"
)

func (p *Provider) ApplyTemplate(name string, filter string, params map[string]interface{}) ([]byte, error) {
	data, err := p.RenderTemplate(name, params)
	if err != nil {
		return nil, err
	}

	return p.Apply(data, filter)
}

func (p *Provider) RenderTemplate(name string, params map[string]interface{}) ([]byte, error) {
	data, err := p.templater.Render(fmt.Sprintf("%s.yml.tmpl", name), params)
	if err != nil {
		return nil, err
	}

	return helpers.FormatYAML(data)
}

func (p *Provider) templateHelpers() template.FuncMap {
	return template.FuncMap{
		"coalesce": func(ss ...string) string {
			return helpers.CoalesceString(ss...)
		},
	}
}
