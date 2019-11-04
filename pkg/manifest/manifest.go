package manifest

import (
	"fmt"
	"io"
	"math/rand"
	"sort"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v2"
)

var (
	DefaultCpu = 256
	DefaultMem = 512
)

type Manifest struct {
	Environment Environment `yaml:"environment,omitempty"`
	Params      Params      `yaml:"params,omitempty"`
	Resources   Resources   `yaml:"resources,omitempty"`
	Services    Services    `yaml:"services,omitempty"`
	Timers      Timers      `yaml:"timers,omitempty"`

	attributes map[string]bool
	env        map[string]string
}

func init() {
	rand.Seed(time.Now().UnixNano())
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

	m.attributes, err = yamlAttributes(p)
	if err != nil {
		return nil, err
	}

	m.env = map[string]string{}

	for k, v := range env {
		m.env[k] = v
	}

	if err := m.ApplyDefaults(); err != nil {
		return nil, err
	}

	if err := m.CombineEnv(); err != nil {
		return nil, err
	}

	if err := m.Validate(); err != nil {
		return nil, err
	}

	return &m, nil
}

func (m *Manifest) Agents() []string {
	a := []string{}

	for _, s := range m.Services {
		if s.Agent.Enabled {
			a = append(a, s.Name)
		}
	}

	return a
}

func (m *Manifest) Attributes() []string {
	attrs := []string{}

	for k := range m.attributes {
		attrs = append(attrs, k)
	}

	sort.Strings(attrs)

	return attrs
}

func (m *Manifest) AttributesByPrefix(prefix string) []string {
	attrs := []string{}

	for _, a := range m.Attributes() {
		if strings.HasPrefix(a, prefix) {
			attrs = append(attrs, a)
		}
	}

	return attrs
}

func (m *Manifest) AttributeSet(name string) bool {
	return m.attributes[name]
}

func (m *Manifest) Env() map[string]string {
	return m.env
}

// used only for tests
func (m *Manifest) SetAttributes(attrs []string) {
	m.attributes = map[string]bool{}

	for _, a := range attrs {
		m.attributes[a] = true
	}
}

// used only for tests
func (m *Manifest) SetEnv(env map[string]string) {
	m.env = env
}

func (m *Manifest) CombineEnv() error {
	for i, s := range m.Services {
		me := make([]string, len(m.Environment))
		copy(me, m.Environment)
		m.Services[i].Environment = append(me, s.Environment...)
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

		return nil, fmt.Errorf("required env: %s", strings.Join(missing, ", "))
	}

	return env, nil
}

func (m *Manifest) Validate() error {
	if err := m.validateEnv(); err != nil {
		return err
	}

	for _, r := range m.Resources {
		if strings.TrimSpace(r.Type) == "" {
			return fmt.Errorf("resource type can not be blank")
		}
	}

	return nil
}

// validateEnv returns an error if required env vars for a service are not available
// It also filters m.env to the union of all service env vars defined in the manifest
func (m *Manifest) validateEnv() error {
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

		if s.Drain == 0 {
			m.Services[i].Drain = 30
		}

		if s.Health.Path == "" {
			m.Services[i].Health.Path = "/"
		}

		if s.Health.Interval == 0 {
			m.Services[i].Health.Interval = 5
		}

		if s.Health.Grace == 0 {
			m.Services[i].Health.Grace = m.Services[i].Health.Interval
		}

		if s.Health.Timeout == 0 {
			m.Services[i].Health.Timeout = m.Services[i].Health.Interval - 1
		}

		if s.Port.Port > 0 && s.Port.Scheme == "" {
			m.Services[i].Port.Scheme = "http"
		}

		sp := fmt.Sprintf("services.%s.scale", s.Name)

		// if no scale attributes set
		if len(m.AttributesByPrefix(sp)) == 0 {
			m.Services[i].Scale.Count = ServiceScaleCount{Min: 1, Max: 1}
		}

		// if no explicit count attribute set yet has multiple scale attributes other than count
		if !m.AttributeSet(fmt.Sprintf("%s.count", sp)) && len(m.AttributesByPrefix(sp)) > 1 {
			m.Services[i].Scale.Count = ServiceScaleCount{Min: 1, Max: 1}
		}

		if m.Services[i].Scale.Cpu == 0 {
			m.Services[i].Scale.Cpu = DefaultCpu
		}

		if m.Services[i].Scale.Memory == 0 {
			m.Services[i].Scale.Memory = DefaultMem
		}

		if !m.AttributeSet(fmt.Sprintf("services.%s.sticky", s.Name)) {
			m.Services[i].Sticky = true
		}
	}

	return nil
}

func message(w io.Writer, format string, args ...interface{}) {
	if w != nil {
		w.Write([]byte(fmt.Sprintf(format, args...) + "\n"))
	}
}
