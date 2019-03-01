package k8s

import (
	"fmt"
	"html/template"
	"sort"
	"strings"

	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/manifest"
	"github.com/convox/rack/pkg/structs"
	shellquote "github.com/kballard/go-shellquote"
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

type envItem struct {
	Key   string
	Value string
}

func (p *Provider) templateHelpers() template.FuncMap {
	return template.FuncMap{
		"domains": func(app string, s manifest.Service) []string {
			ds := []string{
				p.Engine.ServiceHost(app, s),
				fmt.Sprintf("%s.%s.%s.convox", s.Name, app, p.Rack),
			}
			for _, d := range s.Domains {
				ds = append(ds, d)
			}
			return ds
		},
		"env": func(envs ...map[string]string) []envItem {
			env := map[string]string{}
			for _, e := range envs {
				for k, v := range e {
					env[k] = v
				}
			}
			ks := []string{}
			for k := range env {
				ks = append(ks, k)
			}
			sort.Strings(ks)
			eis := []envItem{}
			for _, k := range ks {
				eis = append(eis, envItem{Key: k, Value: env[k]})
			}
			return eis
		},
		"envname": func(s string) string {
			return envName(s)
		},
		// "host": func(app, service string) string {
		//   return p.Engine.ServiceHost(app, service)
		// },
		"image": func(a *structs.App, s manifest.Service, r *structs.Release) (string, error) {
			repo, _, err := p.Engine.RepositoryHost(a.Name)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("%s:%s.%s", repo, s.Name, r.Build), nil
		},
		"safe": func(s string) template.HTML {
			return template.HTML(fmt.Sprintf("%q", s))
		},
		"shellsplit": func(s string) ([]string, error) {
			return shellquote.Split(s)
		},
		"sortedKeys": func(m map[string]string) []string {
			ks := []string{}
			for k := range m {
				ks = append(ks, k)
			}
			sort.Strings(ks)
			return ks
		},
		"systemHost": func() string {
			return p.Engine.SystemHost()
		},
		"systemVolume": func(v string) bool {
			return systemVolume(v)
		},
		"upper": func(s string) string {
			return strings.ToUpper(s)
		},
		"volumeFrom": func(app, service, v string) string {
			return p.volumeFrom(app, service, v)
		},
		"volumeSources": func(app, service string, vs []string) []string {
			return p.volumeSources(app, service, vs)
		},
		"volumeName": func(app, v string) string {
			return p.volumeName(app, v)
		},
		"volumeTo": func(v string) (string, error) {
			return volumeTo(v)
		},
	}
}
