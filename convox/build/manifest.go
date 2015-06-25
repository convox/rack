package build

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	yaml "github.com/convox/cli/Godeps/_workspace/src/gopkg.in/yaml.v2"
)

type Manifest map[string]ManifestEntry

type ManifestEntry struct {
	Build       string      `yaml:"build"`
	Command     interface{} `yaml:"command,omitempty"`
	Environment []string    `yaml:"environment"`
	Ports       []string    `yaml:"ports"`
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
