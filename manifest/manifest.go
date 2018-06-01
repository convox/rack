package manifest

import (
	"fmt"
	"io"
	"sort"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

type Manifest struct {
	Environment Environment `yaml:"environment,omitempty"`
	Resources   Resources   `yaml:"resources,omitempty"`
	Services    Services    `yaml:"services,omitempty"`
	Timers      Timers      `yaml:"timers,omitempty"`

	env map[string]string
}

func Load(data []byte, env map[string]string) (*Manifest, error) {
	var m Manifest

	p, err := interpolate(data, env)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(p, &m); err != nil {
		return nil, err
	}

	m.env = env

	if err := m.ApplyDefaults(); err != nil {
		return nil, err
	}

	if err := m.CombineEnv(); err != nil {
		return nil, err
	}

	if err := m.ValidateEnv(); err != nil {
		return nil, err
	}

	return &m, nil
}

func (m *Manifest) Agents() []string {
	a := []string{}

	for _, s := range m.Services {
		if s.Agent {
			a = append(a, s.Name)
		}
	}

	return a
}

func (m *Manifest) Env() map[string]string {
	return m.env
}

// used only for tests
func (m *Manifest) SetEnv(env map[string]string) {
	m.env = env
}

func (m *Manifest) CombineEnv() error {
	for i, s := range m.Services {
		m.Services[i].Environment = append(m.Environment, s.Environment...)
	}

	return nil
}

func (m *Manifest) Service(name string) (*Service, error) {
	for _, s := range m.Services {
		if s.Name == name {
			return &s, nil
		}
	}

	return nil, fmt.Errorf("no such service: %s", name)
}

func (m *Manifest) ServiceEnvironment(service string) (map[string]string, error) {
	s, err := m.Service(service)
	if err != nil {
		return nil, err
	}

	env := map[string]string{}

	missing := []string{}

	for _, e := range s.Environment {
		parts := strings.SplitN(e, "=", 2)

		switch len(parts) {
		case 1:
			if parts[0] == "*" {
				for k, v := range m.env {
					env[k] = v
				}
			} else {
				v, ok := m.env[parts[0]]
				if !ok {
					missing = append(missing, parts[0])
				}
				env[parts[0]] = v
			}
		case 2:
			v, ok := m.env[parts[0]]
			if ok {
				env[parts[0]] = v
			} else {
				env[parts[0]] = parts[1]
			}
		default:
			return nil, fmt.Errorf("invalid environment declaration: %s", e)
		}
	}

	if len(missing) > 0 {
		sort.Strings(missing)

		return nil, fmt.Errorf("required env: %s\n", strings.Join(missing, ", "))
	}

	return env, nil
}

// ValidateEnv returns an error if required env vars for a service are not available
// It also filters m.env to the union of all service env vars defined in the manifest
func (m *Manifest) ValidateEnv() error {
	keys := map[string]bool{}

	for _, s := range m.Services {
		env, err := m.ServiceEnvironment(s.Name)
		if err != nil {
			return err
		}

		for k := range env {
			keys[k] = true
		}
	}

	for k := range m.env {
		if !keys[k] {
			delete(m.env, k)
		}
	}

	return nil
}

func (m *Manifest) ApplyDefaults() error {
	for i, s := range m.Services {
		if s.Build.Path == "" && s.Image == "" {
			m.Services[i].Build.Path = "."
		}

		if m.Services[i].Build.Path != "" && s.Build.Manifest == "" {
			m.Services[i].Build.Manifest = "Dockerfile"
		}

		if s.Scale.Count == nil {
			m.Services[i].Scale.Count = &ServiceScaleCount{Min: 1, Max: 1}
		}

		if s.Health.Path == "" {
			m.Services[i].Health.Path = "/"
		}

		if s.Health.Interval == 0 {
			m.Services[i].Health.Interval = 5
		}

		if s.Health.Timeout == 0 {
			m.Services[i].Health.Timeout = m.Services[i].Health.Interval - 1
		}

		if s.Scale.Cpu == 0 {
			m.Services[i].Scale.Memory = 256
		}

		if s.Scale.Memory == 0 {
			m.Services[i].Scale.Memory = 512
		}
	}

	return nil
}

func message(w io.Writer, format string, args ...interface{}) {
	if w != nil {
		w.Write([]byte(fmt.Sprintf(format, args...) + "\n"))
	}
}
