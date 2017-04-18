package structs

type Manifest map[string]ManifestEntry

type ManifestV2 struct {
	Version  string
	Services Manifest
}

type ManifestEntry struct {
	Build       string      `yaml:"build,omitempty"`
	Dockerfile  string      `yaml:"dockerfile,omitempty"`
	Image       string      `yaml:"image,omitempty"`
	Command     interface{} `yaml:"command,omitempty"`
	Entrypoint  string      `yaml:"entrypoint,omitempty"`
	Environment interface{} `yaml:"environment,omitempty"`
	Labels      interface{} `yaml:"labels,omitempty"`
	Links       []string    `yaml:"links,omitempty"`
	Ports       interface{} `yaml:"ports,omitempty"`
	Privileged  bool        `yaml:"privileged,omitempty"`
	Volumes     []string    `yaml:"volumes,omitempty"`
}
