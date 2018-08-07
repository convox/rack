package aws

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"net"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"html/template"

	"github.com/convox/rack/manifest"
	"github.com/convox/rack/manifest1"
	"github.com/convox/rack/structs"
)

func formationHelpers() template.FuncMap {
	return template.FuncMap{
		"apex": func(domain string) string {
			parts := strings.Split(domain, ".")
			for i := 0; i < len(parts)-1; i++ {
				d := strings.Join(parts[i:], ".")
				if mx, err := net.LookupMX(d); err == nil && len(mx) > 0 {
					return d
				}
			}
			return domain
		},
		"certificate": func(certs structs.Certificates, domains []string) (string, error) {
			for _, c := range certs {
				found := true
				for _, d := range domains {
					m, err := c.Match(d)
					if err != nil {
						return "", err
					}
					if !m {
						found = false
						break
					}
				}
				if found {
					return c.Arn, nil
				}
			}
			return "", nil
		},
		"dec": func(i int) int {
			return i - 1
		},
		"join": func(ss []string, j string) string {
			return strings.Join(ss, j)
		},
		"priority": func(app, service, domain string, index int) uint32 {
			tier := uint32(1)
			if strings.HasPrefix(domain, "*.") {
				tier = 25000
			}
			return (crc32.ChecksumIEEE([]byte(fmt.Sprintf("%s-%s-%s-%d", app, service, domain, index))) % 25000) + tier
		},
		"router": func(service string, m *manifest.Manifest) (string, error) {
			s, err := m.Service(service)
			if err != nil {
				return "", err
			}
			if s.Internal {
				return "RouterInternal", nil
			}
			return "Router", nil
		},
		"safe": func(s string) template.HTML {
			return template.HTML(fmt.Sprintf("%q", s))
		},
		"services": func(m *manifest.Manifest) string {
			if m == nil {
				return ""
			}
			ss := make([]string, len(m.Services))
			for i, s := range m.Services {
				ss[i] = s.Name
			}
			sort.Strings(ss)
			return strings.Join(ss, ",")
		},
		"statistic": func(s string) (string, error) {
			switch strings.ToLower(s) {
			case "avg":
				return "Average", nil
			case "max":
				return "Maximum", nil
			case "min":
				return "Minimum", nil
			case "sum":
				return "Sum", nil
			}
			return "", fmt.Errorf("unknown metric statistic: %s", s)
		},
		"upcase": func(s string) string {
			return strings.ToUpper(s)
		},
		"upper": func(s string) string {
			return upperName(s)
		},
		"volumeFrom": func(app, s string) string {
			parts := strings.SplitN(s, ":", 2)

			switch v := parts[0]; v {
			case "/var/run/docker.sock":
				return v
			default:
				return path.Join("/volumes", app, v)
			}
		},
		"volumeTo": func(s string) string {
			parts := strings.SplitN(s, ":", 2)
			switch len(parts) {
			case 1:
				return s
			case 2:
				return parts[1]
			}
			return fmt.Sprintf("invalid volume %q", s)
		},
		// generation 1
		"coalesce": func(ss ...string) string {
			for _, s := range ss {
				if s != "" {
					return s
				}
			}
			return ""
		},
		"itoa": func(i int) string {
			return strconv.Itoa(i)
		},
		"value": func(s string) template.HTML {
			return template.HTML(fmt.Sprintf("%q", s))
		},
		"agents": func(m *manifest1.Manifest) string {
			if m == nil {
				return ""
			}
			as := []string{}
			for _, s := range m.Services {
				if s.IsAgent() {
					as = append(as, s.Name)
				}
			}
			sort.Strings(as)
			return strings.Join(as, ",")
		},
		"cronjobs": func(a *structs.App, m *manifest1.Manifest) CronJobs {
			return appCronJobs(a, m)
		},
	}
}
func formationTemplate(name string, data interface{}) ([]byte, error) {
	var buf bytes.Buffer

	path := fmt.Sprintf("provider/aws/formation/%s.json.tmpl", name)
	file := filepath.Base(path)

	t, err := template.New(file).Funcs(formationHelpers()).ParseFiles(path)
	if err != nil {
		return nil, err
	}

	if err := t.Execute(&buf, data); err != nil {
		return nil, err
	}

	var v interface{}

	if err := json.Unmarshal(buf.Bytes(), &v); err != nil {
		switch t := err.(type) {
		case *json.SyntaxError:
			return nil, jsonSyntaxError(t, buf.Bytes())
		}
		return nil, err
	}

	return json.MarshalIndent(v, "", "  ")
}

func jsonSyntaxError(err *json.SyntaxError, data []byte) error {
	start := bytes.LastIndex(data[:err.Offset], []byte("\n")) + 1
	line := bytes.Count(data[:start], []byte("\n"))
	pos := int(err.Offset) - start - 1
	ltext := strings.Split(string(data), "\n")[line]

	return fmt.Errorf("json syntax error: line %d pos %d: %s: %s", line, pos, err.Error(), ltext)
}
