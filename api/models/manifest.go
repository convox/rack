package models

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

type Manifest []ManifestEntry

type ManifestEntry struct {
	Name string

	Build   string      `yaml:"build"`
	Command interface{} `yaml:"command"`
	Env     []string    `yaml:"env"`
	Image   string      `yaml:"image"`
	Links   []string    `yaml:"links"`
	Ports   []string    `yaml:"ports"`
	Volumes []string    `yaml:"volumes"`

	randoms []int
}

type ManifestEntries map[string]ManifestEntry

func LoadManifest(data string) (Manifest, error) {
	var entries ManifestEntries

	err := yaml.Unmarshal([]byte(data), &entries)

	if err != nil {
		return nil, err
	}

	manifest := make(Manifest, 0)

	currentPort := 5000

	for name, entry := range entries {
		entry.Name = name

		for _ = range entry.Ports {
			entry.randoms = append(entry.randoms, currentPort)
			currentPort += 1
		}

		manifest = append(manifest, ManifestEntry(entry))
	}

	return manifest, nil
}

func (m Manifest) Entry(name string) *ManifestEntry {
	for _, me := range m {
		if me.Name == name {
			return &me
		}
	}

	return nil
}

func (m Manifest) EntryNames() []string {
	names := make([]string, len(m))

	for i, entry := range m {
		names[i] = entry.Name
	}

	return names
}

func (m Manifest) HasPorts() bool {
	if len(m) == 0 {
		return true // special case to pre-initialize ELB at app create
	}

	for _, me := range m {
		if len(me.Ports) > 0 {
			return true
		}
	}

	return false
}

func (m Manifest) HasProcesses() bool {
	return len(m) > 0
}

func (me *ManifestEntry) CommandString() string {
	switch cmd := me.Command.(type) {
	case nil:
		return ""
	case string:
		return cmd
	case []interface{}:
		parts := make([]string, len(cmd))

		for i, c := range cmd {
			parts[i] = c.(string)
		}

		return strings.Join(parts, " ")
	default:
		fmt.Fprintf(os.Stderr, "unexpected type for command: %T\n", cmd)
		return ""
	}
}

func (me ManifestEntry) HasPorts() bool {
	return len(me.Ports) > 0
}

func (me *ManifestEntry) Randoms() []int {
	return me.randoms
}
