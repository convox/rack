package models

import (
	"strings"

	"github.com/convox/kernel/web/Godeps/_workspace/src/gopkg.in/yaml.v2"
)

type Manifest map[string]ManifestEntry

type ManifestEntry struct {
	Build string   `yaml:"build"`
	Env   []string `yaml:"env"`
	Image string   `yaml:"image"`
	Links []string `yaml:"links"`
}

func LoadManifest(data string) (*Manifest, error) {
	var manifest *Manifest

	err := yaml.Unmarshal([]byte(data), &manifest)

	if err != nil {
		return nil, err
	}

	return manifest, nil
}

func (m *Manifest) Apply(app *App) error {
	original, err := ListProcesses(app.Name)

	if err != nil {
		return err
	}

	processes := make(Processes, 0)

	for name, entry := range *m {
		if strings.HasPrefix(entry.Image, "convox/") {
		} else {
			count := "1"

			for _, p := range original {
				if p.Name == name {
					count = p.Count
				}
			}

			process := &Process{
				Name:  name,
				Count: count,
				App:   app.Name,
			}

			process.Save()

			processes = append(processes, *process)
		}
	}

	app.Processes = processes

	return nil
}
