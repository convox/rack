package manifest

import (
	"fmt"
	"io"
	"math/rand"
	"net"
	"regexp"
	"sort"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v2"
)

const (
	ValidNameDescription = "must contain only lowercase alphanumeric and dashes"
)

var (
	nameValidator = regexp.MustCompile(`^[a-z]{1}[a-z0-9-]*$`)

	// nlbCertARNPattern accepts ACM certificate ARNs and IAM server-certificate
	// ARNs across all AWS partitions (aws, aws-cn, aws-us-gov).
	nlbCertARNPattern = regexp.MustCompile(
		`^arn:aws[-a-z]*:acm:[a-z0-9-]+:\d{12}:certificate/[a-zA-Z0-9-]+$|` +
			`^arn:aws[-a-z]*:iam::\d{12}:server-certificate/.+$`,
	)
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

	nlbPortOwner := map[int]string{}
	for _, s := range m.Services {
		if !nameValidator.MatchString(s.Name) {
			return fmt.Errorf("service name %s invalid, %s", s.Name, ValidNameDescription)
		}

		if len(s.NLB) > 0 && s.Agent.Enabled {
			return fmt.Errorf("service %s: agent mode is incompatible with nlb ports", s.Name)
		}

		seenPorts := map[int]int{}
		seenContainerPorts := map[int]bool{}
		for _, np := range s.NLB {
			if np.Port < 1 || np.Port > 65535 {
				return fmt.Errorf("service %s: nlb port %d out of range", s.Name, np.Port)
			}
			if np.ContainerPort < 1 || np.ContainerPort > 65535 {
				return fmt.Errorf("service %s: nlb containerPort %d out of range", s.Name, np.ContainerPort)
			}
			if np.Protocol != "tcp" && np.Protocol != "tls" {
				return fmt.Errorf("service %s nlb port %d: protocol must be tcp or tls, got %q", s.Name, np.Port, np.Protocol)
			}
			if np.Protocol == "tls" && np.Certificate == "" {
				return fmt.Errorf("service %s nlb port %d: protocol tls requires a certificate (provide an ACM ARN; run 'convox certs list' to see available ARNs)", s.Name, np.Port)
			}
			if np.Protocol != "tls" && np.Certificate != "" {
				return fmt.Errorf("service %s nlb port %d: certificate is only valid with protocol: tls", s.Name, np.Port)
			}
			if np.Certificate != "" && !nlbCertARNPattern.MatchString(np.Certificate) {
				return fmt.Errorf("service %s nlb port %d: certificate must be a full ACM or IAM server-certificate ARN, got %q (run 'convox certs list' to see available ARNs)", s.Name, np.Port, np.Certificate)
			}
			if np.Scheme != "public" && np.Scheme != "internal" {
				return fmt.Errorf("service %s nlb port %d: scheme must be public or internal, got %q", s.Name, np.Port, np.Scheme)
			}
			seenAllowCIDR := map[string]bool{}
			for _, c := range np.AllowCIDR {
				ip, ipnet, err := net.ParseCIDR(c)
				if err != nil {
					return fmt.Errorf("service %s nlb port %d: allow_cidr entry %q is not a valid IPv4 CIDR (e.g. 10.0.0.0/16)", s.Name, np.Port, c)
				}
				if ip.To4() == nil {
					return fmt.Errorf("service %s nlb port %d: allow_cidr entry %q is not a valid IPv4 CIDR (IPv6 not supported)", s.Name, np.Port, c)
				}
				if ipnet.String() != c {
					return fmt.Errorf("service %s nlb port %d: allow_cidr entry %q is not canonical; use %q instead", s.Name, np.Port, c, ipnet.String())
				}
				if seenAllowCIDR[c] {
					return fmt.Errorf("service %s nlb port %d: allow_cidr contains duplicate entry: %s", s.Name, np.Port, c)
				}
				seenAllowCIDR[c] = true
			}
			if existing, ok := seenPorts[np.Port]; ok {
				if existing != np.ContainerPort {
					return fmt.Errorf("service %s: nlb port %d declared with conflicting containerPort values", s.Name, np.Port)
				}
				return fmt.Errorf("service %s: duplicate nlb port %d", s.Name, np.Port)
			}
			if seenContainerPorts[np.ContainerPort] {
				return fmt.Errorf("service %s: nlb containerPort %d used by multiple nlb listeners", s.Name, np.ContainerPort)
			}
			if owner, ok := nlbPortOwner[np.Port]; ok && owner != s.Name {
				return fmt.Errorf("nlb port %d declared by services %s and %s; rack NLB listener is shared, each port must be unique across services", np.Port, owner, s.Name)
			}
			nlbPortOwner[np.Port] = s.Name
			seenPorts[np.Port] = np.ContainerPort
			seenContainerPorts[np.ContainerPort] = true
		}
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

		if !m.AttributeSet(fmt.Sprintf("services.%s.deployment.maximum", s.Name)) {
			if s.Agent.Enabled || s.Singleton {
				m.Services[i].Deployment.Maximum = 100
			} else {
				m.Services[i].Deployment.Maximum = 200
			}
		}

		if !m.AttributeSet(fmt.Sprintf("services.%s.deployment.minimum", s.Name)) {
			if s.Agent.Enabled || s.Singleton {
				m.Services[i].Deployment.Minimum = 0
			} else {
				m.Services[i].Deployment.Minimum = 50
			}
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
			m.Services[i].Scale.Cooldown = ServiceScaleCooldown{Down: 60, Up: 60}
		}

		// if no explicit count attribute set yet has multiple scale attributes other than count
		if !m.AttributeSet(fmt.Sprintf("%s.count", sp)) && len(m.AttributesByPrefix(sp)) > 1 {
			m.Services[i].Scale.Count = ServiceScaleCount{Min: 1, Max: 1}
		}

		// if no explicit cooldown attribute set yet has scale attributes other than cooldown
		if !m.AttributeSet(fmt.Sprintf("%s.cooldown", sp)) && len(m.AttributesByPrefix(sp)) >= 1 {
			m.Services[i].Scale.Cooldown = ServiceScaleCooldown{Down: 60, Up: 60}
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

		if !m.AttributeSet(fmt.Sprintf("services.%s.termination.grace", s.Name)) {
			m.Services[i].Termination.Grace = 30
		}

		if s.InternalAndExternal {
			m.Services[i].Internal = true
		}

		for j := range m.Services[i].NLB {
			np := &m.Services[i].NLB[j]
			np.Protocol = strings.ToLower(strings.TrimSpace(np.Protocol))
			if np.Protocol == "" {
				np.Protocol = "tcp"
			}
			np.Scheme = strings.ToLower(strings.TrimSpace(np.Scheme))
			if np.Scheme == "" {
				np.Scheme = "public"
			}
			if np.ContainerPort == 0 {
				np.ContainerPort = np.Port
			}
		}
	}

	return nil
}

func message(w io.Writer, format string, args ...interface{}) {
	if w != nil {
		w.Write([]byte(fmt.Sprintf(format, args...) + "\n"))
	}
}
