package structs

import (
	"html/template"

	"github.com/convox/rack/Godeps/_workspace/src/gopkg.in/yaml.v2"
)

type Manifest []ManifestEntry

type ManifestEntry struct {
	Name string

	Build      string                   `yaml:"build"`
	Command    interface{}              `yaml:"command"`
	Env        []string                 `yaml:"environment"`
	Exports    map[string]string        `yaml:"-"`
	Image      string                   `yaml:"image"`
	Links      []string                 `yaml:"links"`
	LinkVars   map[string]template.HTML `yaml:"-"`
	Ports      []string                 `yaml:"ports"`
	Privileged bool                     `yaml:"privileged"`
	Volumes    []string                 `yaml:"volumes"`

	primary bool
	randoms map[string]int
}

func LoadManifest(manifest string) (*Manifest, error) {
	var m Manifest

	err := yaml.Unmarshal([]byte(manifest), &m)

	if err != nil {
		return nil, err
	}

	return &m, nil
}

func (m Manifest) Entry(name string) *ManifestEntry {
	for _, me := range m {
		if me.Name == name {
			return &me
		}
	}

	return nil
}
