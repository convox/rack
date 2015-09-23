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
	Env     []string    `yaml:"environment"`
	Image   string      `yaml:"image"`
	Links   []string    `yaml:"links"`
	Ports   []string    `yaml:"ports"`
	Volumes []string    `yaml:"volumes"`

	randoms map[string]int
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
		entry.randoms = make(map[string]int)

		for _, port := range entry.Ports {
			entry.randoms[port] = currentPort
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

func (m Manifest) Formation() (string, error) {
	data, err := buildTemplate("app", "app", m)

	if err != nil {
		return "", err
	}

	pretty, err := prettyJson(string(data))

	if err != nil {
		return "", err
	}

	return pretty, nil
}

func (m Manifest) HasExternalPorts() bool {
	if len(m) == 0 {
		return true // special case to pre-initialize ELB at app create
	}

	for _, me := range m {
		if len(me.ExternalPorts()) > 0 {
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

func (me ManifestEntry) InternalPorts() []string {
	internal := []string{}

	for _, port := range me.Ports {
		if len(strings.Split(port, ":")) == 1 {
			internal = append(internal, port)
		}
	}

	return internal
}

func (me ManifestEntry) ExternalPorts() []string {
	ext := []string{}

	for _, port := range me.Ports {
		if len(strings.Split(port, ":")) == 2 {
			ext = append(ext, port)
		}
	}

	return ext
}

func (me ManifestEntry) Randoms() map[string]int {
	return me.randoms
}
