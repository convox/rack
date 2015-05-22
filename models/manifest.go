package models

import (
	"fmt"
	"os"
	"strconv"
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
			ps := Process{
				Name:    entry.Name,
				Command: entry.Command,
				Count:   1,
			}

			for _, p := range entry.Ports {
				pp := strings.Split(p, ":")
				sp := pp[0]

				if len(pp) > 1 {
					sp = pp[1]
				}

				port, err := strconv.Atoi(sp)

				if err != nil {
					fmt.Fprintf(os.Stderr, "error: %s\n", err)
					continue
				}

				ps.Ports = append(ps.Ports, port)
			}

			processes = append(processes, ps)
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
