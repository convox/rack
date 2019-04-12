package k8s

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"os/exec"
	"sort"
	"strings"

	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/manifest"
	"github.com/convox/rack/pkg/structs"
	shellquote "github.com/kballard/go-shellquote"
	yaml "gopkg.in/yaml.v2"
)

func (p *Provider) Apply(data []byte, filter string) ([]byte, error) {
	rs, err := templateResources(filter)
	if err != nil {
		return nil, err
	}

	labels := parseLabels(filter)

	parts := bytes.Split(data, []byte("---\n"))

	for i := range parts {
		dp, err := applyLabels(parts[i], labels)
		if err != nil {
			panic(err)
			return nil, err
		}

		parts[i] = dp
	}

	data = bytes.Join(parts, []byte("---\n"))

	args := []string{"--prune", "-l", filter}

	for _, r := range rs {
		args = append(args, "--prune-whitelist", r)
	}

	out, err := kubectlApply(data, args...)
	if err != nil {
		if !strings.Contains(string(out), "is immutable") {
			return out, err
		}

		out, err := kubectlApply(data, "--force")
		if err != nil {
			return out, err
		}
	}

	return out, nil
}

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
		"base64": func(s string) string {
			return string(base64.StdEncoding.EncodeToString([]byte(s)))
		},
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
		"image": func(a *structs.App, s manifest.Service, r *structs.Release) (string, error) {
			repo, _, err := p.Engine.RepositoryHost(a.Name)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("%s:%s.%s", repo, s.Name, r.Build), nil
		},
		"lower": func(s string) string {
			return strings.ToLower(s)
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

func applyLabels(data []byte, labels map[string]string) ([]byte, error) {
	var v map[string]interface{}

	if err := yaml.Unmarshal(data, &v); err != nil {
		return nil, err
	}

	if len(v) == 0 {
		return data, nil
	}

	switch t := v["metadata"].(type) {
	case nil:
		v["metadata"] = map[string]interface{}{"labels": labels}
	case map[interface{}]interface{}:
		switch u := t["labels"].(type) {
		case nil:
			t["labels"] = labels
			v["metadata"] = t
		case map[interface{}]interface{}:
			for k, v := range labels {
				u[k] = v
			}
			t["labels"] = u
			v["metadata"] = t
		default:
			return nil, fmt.Errorf("unknown labels type: %T", u)
		}
	default:
		return nil, fmt.Errorf("unknown metadata type: %T", t)
	}

	pd, err := yaml.Marshal(v)
	if err != nil {
		return nil, err
	}

	return pd, nil
}

func kubectlApply(data []byte, args ...string) ([]byte, error) {
	ka := append([]string{"apply", "-f", "-"}, args...)

	cmd := exec.Command("kubectl", ka...)

	cmd.Stdin = bytes.NewReader(data)

	return cmd.CombinedOutput()
}

func parseLabels(labels string) map[string]string {
	ls := map[string]string{}

	for _, part := range strings.Split(labels, ",") {
		ps := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(ps) == 2 {
			ls[ps[0]] = ps[1]
		}
	}

	return ls
}

func templateResources(filter string) ([]string, error) {
	data, err := exec.Command("kubectl", "api-resources", "--verbs=list", "--namespaced", "-o", "name").CombinedOutput()
	if err != nil {
		return []string{}, nil
	}

	ars := strings.Split(strings.TrimSpace(string(data)), "\n")

	rsh := map[string]bool{}

	data, err = exec.Command("kubectl", "get", "-l", filter, "--all-namespaces", "-o", "json", strings.Join(ars, ",")).CombinedOutput()
	if err != nil {
		return []string{}, nil
	}

	if strings.TrimSpace(string(data)) == "" {
		return []string{}, nil
	}

	var res struct {
		Items []struct {
			ApiVersion string `json:"apiVersion"`
			Kind       string `json:"kind"`
		}
	}

	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}

	for _, i := range res.Items {
		av := i.ApiVersion

		if !strings.Contains(av, "/") {
			av = fmt.Sprintf("core/%s", av)
		}

		rsh[fmt.Sprintf("%s/%s", av, i.Kind)] = true
	}

	rs := []string{}

	for r := range rsh {
		rs = append(rs, r)
	}

	sort.Strings(rs)

	return rs, nil
}
