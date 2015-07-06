package build

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	yaml "github.com/convox/build/Godeps/_workspace/src/gopkg.in/yaml.v2"
)

type Manifest map[string]ManifestEntry

type ManifestEntry struct {
	Build       string      `yaml:"build,omitempty"`
	Image       string      `yaml:"image,omitempty"`
	Command     interface{} `yaml:"command,omitempty"`
	Environment []string    `yaml:"environment"`
	Links       []string    `yaml:"links,omitempty"`
	Ports       []string    `yaml:"ports"`
	Volumes     []string    `yaml:"volumes,omitempty"`
}

func (m Manifest) Tags(registry string, project string, tag string) map[string]string {
	tags := make(map[string]string)

	for key := range m {
		ps := m[key]

		img := ps.Image

		if img == "" {
			img = fmt.Sprintf("%s_%s", project, key)
		}

		tags[fmt.Sprintf("%s/%s-%s:%s", registry, project, key, tag)] = img
	}

	return tags
}

func (m Manifest) String() string {
	b, _ := yaml.Marshal(m)
	return string(b)
}

func ManifestFromInspect(data []byte) ([]byte, error) {
	var exposed map[string]interface{}

	err := json.Unmarshal(data, &exposed)

	if err != nil {
		return nil, err
	}

	// sort exposed numerically
	e := make([]string, len(exposed))
	i := 0
	for k, _ := range exposed {
		e[i] = k
		i++
	}

	sort.Strings(e)

	var ports []string

	cur := 5000

	for i := range e {
		port := e[i]
		ports = append(ports, fmt.Sprintf("%d:%s", cur, strings.Split(port, "/")[0]))
		cur += 100
	}

	manifest := make(Manifest)

	entry := ManifestEntry{
		Build: ".",
		Ports: ports,
	}

	manifest["web"] = entry

	return yaml.Marshal(manifest)
}

func ManifestFromProcfile(procs map[string]string) ([]byte, error) {
	manifest := make(Manifest)

	for name, command := range procs {
		entry := ManifestEntry{
			Build:   ".",
			Command: command,
		}

		if name == "web" {
			entry.Ports = []string{"5000:3000"}
		}

		manifest[name] = entry
	}

	return yaml.Marshal(manifest)
}

func ManifestFromBytes(b []byte) (Manifest, error) {
	m := make(Manifest)
	err := yaml.Unmarshal(b, &m)
	return m, err
}
