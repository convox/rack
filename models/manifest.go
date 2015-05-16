package models

import (
	"strings"

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
		if st := entry.ServiceType(); st == "" {
			processes = append(processes, Process{
				Name:    entry.Name,
				Command: entry.Command,
				Count:   1,
			})
		}
	}

	return processes
}

func (m *Manifest) Services() Services {
	services := Services{}

	for _, entry := range *m {
		if st := entry.ServiceType(); st != "" {
			services = append(services, Service{
				Name: entry.Name,
				Type: st,
			})
		}
	}

	return services
}

func (me *ManifestEntry) ServiceType() string {
	if strings.HasPrefix(me.Image, "convox/") {
		return me.Image
	} else {
		return ""
	}
}
