package models

import (
	"github.com/convox/kernel/Godeps/_workspace/src/gopkg.in/yaml.v2"
)

type Manifest []ManifestEntry

type ManifestEntry struct {
	Name string

	Build   string   `yaml:"build"`
	Command string   `yaml:"command"`
	Env     []string `yaml:"env"`
	Image   string   `yaml:"image"`
	Links   []string `yaml:"links"`
	Ports   []string `yaml:"ports"`
}

type ManifestEntries map[string]ManifestEntry

func LoadManifest(data string) (Manifest, error) {
	var entries ManifestEntries

	err := yaml.Unmarshal([]byte(data), &entries)

	if err != nil {
		return nil, err
	}

	manifest := make(Manifest, 0)

	for name, entry := range entries {
		entry.Name = name
		manifest = append(manifest, ManifestEntry(entry))
	}

	return manifest, nil
}

func (m *Manifest) Processes() Processes {
	processes := Processes{}

	for _, entry := range *m {
		ps := Process{
			Name:    entry.Name,
			Command: entry.Command,
			Count:   1,
		}

		processes = append(processes, ps)
	}

	return processes
}
