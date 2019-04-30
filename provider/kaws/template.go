package kaws

import (
	"fmt"
	"html/template"

	"github.com/convox/rack/pkg/helpers"
)

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
		"safe": func(s string) template.HTML {
			return template.HTML(fmt.Sprintf("%q", s))
		},
		"upper": func(s string) string {
			return upperName(s)
		},
	}
}
