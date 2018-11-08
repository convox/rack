package klocal

import (
	"fmt"

	"github.com/convox/rack/pkg/manifest"
)

func (p *Provider) ResourceRender(app string, r manifest.Resource) ([]byte, error) {
	params := map[string]interface{}{
		"App":        app,
		"Namespace":  p.AppNamespace(app),
		"Name":       r.Name,
		"Parameters": r.Options,
		"Password":   "foo",
		"Rack":       p.Rack,
	}

	return p.RenderTemplate(fmt.Sprintf("resource/%s", r.Type), params)
}
