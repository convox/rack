package structs

import (
	"html/template"
	"math/rand"
	"sort"
	"strings"

	"github.com/convox/rack/Godeps/_workspace/src/gopkg.in/yaml.v2"
)

var (
	ManifestRandomPorts = true
)

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

type ManifestEntries map[string]ManifestEntry
type Manifest []ManifestEntry

func LoadManifest(data []byte) (*Manifest, error) {
	var entries ManifestEntries

	err := yaml.Unmarshal(data, &entries)

	if err != nil {
		return nil, err
	}

	names := []string{}

	for name, _ := range entries {
		names = append(names, name)
	}

	sort.Strings(names)

	manifest := Manifest{}

	currentPort := 5000

	for _, name := range names {
		entry := entries[name]
		entry.Name = name
		entry.randoms = make(map[string]int)

		for _, port := range entry.Ports {
			p := strings.Split(port, ":")[0]

			if ManifestRandomPorts {
				entry.randoms[p] = rand.Intn(62000) + 3000
			} else {
				entry.randoms[p] = currentPort
				currentPort += 1
			}
		}

		manifest = append(manifest, ManifestEntry(entry))
	}

	return &manifest, nil
}

func (m Manifest) Entry(name string) *ManifestEntry {
	for _, me := range m {
		if me.Name == name {
			return &me
		}
	}

	return nil
}
