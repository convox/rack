package models

import (
	"strings"

	"github.com/convox/kernel/web/Godeps/_workspace/src/gopkg.in/yaml.v2"
)

type Manifest []ManifestEntry

type ManifestEntry struct {
	Build string   `yaml:"build"`
	Env   []string `yaml:"env"`
	Image string   `yaml:"image"`
	Links []string `yaml:"links"`
	Name  string
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

func (m *Manifest) Apply(app *App) error {
	original, err := ListProcesses(app.Name)

	if err != nil {
		return err
	}

	for _, entry := range *m {
		if st := entry.ServiceType(); st != "" {
			service := Service{
				Name: entry.Name,
				Type: st,
				App:  app.Name,
			}

			service.Save()
		} else {
			count := "1"

			for _, p := range original {
				if p.Name == entry.Name {
					count = p.Count
				}
			}

			process := &Process{
				Name:  entry.Name,
				Count: count,
				App:   app.Name,
			}

			process.Save()
		}
	}

	return nil
}

func (me *ManifestEntry) ServiceType() string {
	if strings.HasPrefix(me.Image, "convox/") {
		return me.Image
	} else {
		return ""
	}
}
