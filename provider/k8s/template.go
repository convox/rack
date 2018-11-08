package k8s

import (
	"crypto/sha256"
	"fmt"
	"html/template"
	"path"
	"sort"
	"strings"

	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/manifest"
	"github.com/convox/rack/pkg/structs"
	shellquote "github.com/kballard/go-shellquote"
)

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
			return strings.Replace(strings.ToUpper(s), "-", "_", -1)
		},
		"host": func(app, service string) string {
			return p.Engine.ServiceHost(app, service)
		},
		"image": func(a *structs.App, s manifest.Service, r *structs.Release) (string, error) {
			repo, _, err := p.Engine.AppRepository(a.Name)
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
		"upper": func(s string) string {
			return strings.ToUpper(s)
		},
		"volumeFrom": func(app, v string) string {
			if from := strings.Split(v, ":")[0]; systemVolume(from) {
				return from
			} else {
				return path.Join("/mnt/volumes", app, from)
			}
		},
		"volumeName": func(v string) string {
			hash := sha256.Sum256([]byte(v))
			return fmt.Sprintf("volume-%x", hash[0:20])
		},
		"volumeTo": func(v string) (string, error) {
			switch parts := strings.SplitN(v, ":", 2); len(parts) {
			case 1:
				return parts[0], nil
			case 2:
				return parts[1], nil
			default:
				return "", fmt.Errorf("invalid volume %q", v)
			}
		},
	}
}
