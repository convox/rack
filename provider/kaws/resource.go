package kaws

import (
	"crypto/sha256"
	"fmt"

	"github.com/convox/rack/pkg/manifest"
)

func (p *Provider) ResourceRender(app string, r manifest.Resource) ([]byte, error) {
	params := map[string]interface{}{
		"App":        app,
		"Namespace":  p.AppNamespace(app),
		"Name":       r.Name,
		"Parameters": r.Options,
		"Password":   fmt.Sprintf("%x", sha256.Sum256([]byte(p.StackId)))[0:30],
		"Rack":       p.Rack,
	}

	return p.RenderTemplate(fmt.Sprintf("resource/%s", r.Type), params)
}
