package k8s

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"html/template"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/convox/rack/pkg/manifest"
	"github.com/convox/rack/pkg/structs"
	shellquote "github.com/kballard/go-shellquote"
	yaml "gopkg.in/yaml.v2"
)

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
		"host": func(app, service string) string {
			return p.HostFunc(app, service)
		},
		"image": func(a *structs.App, s manifest.Service, r *structs.Release) (string, error) {
			repo, _, err := p.RepoFunc(a.Name)
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

func (p *Provider) yamlTemplate(name string, params interface{}) ([]byte, error) {
	var buf bytes.Buffer

	path := fmt.Sprintf("provider/k8s/template/%s.yml.tmpl", name)
	file := filepath.Base(path)

	t, err := template.New(file).Funcs(p.templateHelpers()).ParseFiles(path)
	if err != nil {
		return nil, err
	}

	if err := t.Execute(&buf, params); err != nil {
		return nil, err
	}

	// fmt.Printf("buf.String() = %+v\n", buf.String())

	parts := bytes.Split(buf.Bytes(), []byte("---"))

	for i, part := range parts {
		var v interface{}

		if err := yaml.Unmarshal(part, &v); err != nil {
			return nil, err
		}

		data, err := yaml.Marshal(v)
		if err != nil {
			return nil, err
		}

		parts[i] = data
	}

	return bytes.Join(parts, []byte("---\n")), nil
}
