package manifest

import (
	"crypto/sha1"
	"fmt"
	"sort"
	"strings"
)

type Service struct {
	Name string `yaml:"-"`

	Agent       bool           `yaml:"agent,omitempty"`
	Build       ServiceBuild   `yaml:"build,omitempty"`
	Command     string         `yaml:"command,omitempty"`
	Domains     ServiceDomains `yaml:"domain,omitempty"`
	Environment Environment    `yaml:"environment,omitempty"`
	Health      ServiceHealth  `yaml:"health,omitempty"`
	Image       string         `yaml:"image,omitempty"`
	Internal    bool           `yaml:"internal,omitempty"`
	Links       []string       `yaml:"links,omitempty"`
	Port        ServicePort    `yaml:"port,omitempty"`
	Privileged  bool           `yaml:"privileged,omitempty"`
	Resources   []string       `yaml:"resources,omitempty"`
	Scale       ServiceScale   `yaml:"scale,omitempty"`
	Test        string         `yaml:"test,omitempty"`
	Volumes     []string       `yaml:"volumes,omitempty"`
}

type Services []Service

type ServiceBuild struct {
	Args     []string `yaml:"args,omitempty"`
	Manifest string   `yaml:"manifest,omitempty"`
	Path     string   `yaml:"path,omitempty"`
}

type ServiceCommand struct {
	Development string
	Test        string
	Production  string
}

type ServiceDomains []string

// type ServiceEnvironment []string

type ServiceHealth struct {
	Grace    int
	Interval int
	Path     string
	Timeout  int
}

type ServicePort struct {
	Port   int    `yaml:"port,omitempty"`
	Scheme string `yaml:"scheme,omitempty"`
}

type ServiceScale struct {
	Count   *ServiceScaleCount
	Cpu     int
	Memory  int
	Targets ServiceScaleTargets `yaml:"targets,omitempty"`
}

type ServiceScaleCount struct {
	Min int
	Max int
}

type ServiceScaleMetric struct {
	Aggregate  string
	Dimensions map[string]string
	Namespace  string
	Name       string
	Value      float64
}

type ServiceScaleMetrics []ServiceScaleMetric

type ServiceScaleTargets struct {
	Cpu      int
	Custom   ServiceScaleMetrics
	Memory   int
	Requests int
}

func (s Service) BuildHash() string {
	return fmt.Sprintf("%x", sha1.Sum([]byte(fmt.Sprintf("build[path=%q, manifest=%q, args=%v] image=%q", s.Build.Path, s.Build.Manifest, s.Build.Args, s.Image))))
}

func (s Service) Domain() string {
	if len(s.Domains) < 1 {
		return ""
	}

	return s.Domains[0]
}

func (s Service) EnvironmentDefaults() map[string]string {
	defaults := map[string]string{}

	for _, e := range s.Environment {
		switch parts := strings.Split(e, "="); len(parts) {
		case 2:
			defaults[parts[0]] = parts[1]
		}
	}

	return defaults
}

func (s Service) EnvironmentKeys() string {
	keys := make([]string, len(s.Environment))

	for i, e := range s.Environment {
		keys[i] = strings.Split(e, "=")[0]
	}

	sort.Strings(keys)

	return strings.Join(keys, ",")
}

func (s Service) GetName() string {
	return s.Name
}

func (s *Service) SetDefaults() error {
	if s.Scale.Count == nil {
		s.Scale.Count = &ServiceScaleCount{Min: 1, Max: 1}
	}

	if s.Scale.Cpu == 0 {
		s.Scale.Cpu = 256
	}

	if s.Scale.Memory == 0 {
		s.Scale.Memory = 512
	}

	return nil
}

func (s ServiceScale) Autoscale() bool {
	switch {
	case s.Targets.Cpu > 0:
		return true
	case len(s.Targets.Custom) > 0:
		return true
	case s.Targets.Memory > 0:
		return true
	case s.Targets.Requests > 0:
		return true
	}

	return false
}
