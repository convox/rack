package manifest

import (
	"crypto/sha1"
	"fmt"
	"strings"
)

type Service struct {
	Name string `yaml:"-"`

	Build       ServiceBuild       `yaml:"build,omitempty"`
	Certificate string             `yaml:"certificate,omitempty"`
	Command     string             `yaml:"command,omitempty"`
	Environment ServiceEnvironment `yaml:"environment,omitempty"`
	Health      ServiceHealth      `yaml:"health,omitempty"`
	Image       string             `yaml:"image,omitempty"`
	Port        ServicePort        `yaml:"port,omitempty"`
	Resources   []string           `yaml:"resources,omitempty"`
	Scale       ServiceScale       `yaml:"scale,omitempty"`
	Test        string             `yaml:"test,omitempty"`
	Volumes     []string           `yaml:"volumes,omitempty"`
}

type Services []Service

type ServiceBuild struct {
	Args []string `yaml:"args,omitempty"`
	Path string   `yaml:"path,omitempty"`
}

type ServiceCommand struct {
	Development string
	Test        string
	Production  string
}

type ServiceEnvironment []string

type ServiceHealth struct {
	Interval int
	Path     string
	Timeout  int
}

type ServicePort struct {
	Port   int
	Scheme string
}

type ServiceScale struct {
	Count  *ServiceScaleCount
	Memory int
}

type ServiceScaleCount struct {
	Min int
	Max int
}

func (s Service) BuildHash() string {
	return fmt.Sprintf("%x", sha1.Sum([]byte(fmt.Sprintf("build[path=%q, args=%v] image=%q", s.Build.Path, s.Build.Args, s.Image))))
}

func (s Service) GetName() string {
	return s.Name
}

func (s Service) ResolvedEnvironment(env map[string]string) map[string]string {
	resolved := map[string]string{}

	for _, e := range s.Environment {
		parts := strings.Split(e, "=")

		switch len(parts) {
		case 1:
			resolved[parts[0]] = env[parts[0]]
		case 2:
			resolved[parts[0]] = coalesce(env[parts[0]], parts[1])
		}
	}

	return resolved
}

func (s *Service) SetDefaults() error {
	if s.Scale.Count == nil {
		s.Scale.Count = &ServiceScaleCount{Min: 1, Max: 1}
	}

	return nil
}
