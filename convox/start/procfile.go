package start

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"

	yaml "github.com/convox/cli/Godeps/_workspace/src/gopkg.in/yaml.v2"
)

type Manifest map[string]ManifestEntry

type ManifestEntry struct {
	Build       string      `yaml:"build"`
	Command     interface{} `yaml:"command,omitempty"`
	Environment []string    `yaml:"environment"`
	Ports       []string    `yaml:"ports"`
}

var procfileEntryRegexp = regexp.MustCompile("^([A-Za-z0-9_]+):\\s*(.+)$")

func Procfile(base string) error {
	data, err := ioutil.ReadFile(filepath.Join(base, "Procfile"))

	if err != nil {
		return err
	}

	procfile, err := parseProcfile(data)

	if err != nil {
		return err
	}

	data, err = genDockerCompose(procfile)

	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(base, "docker-compose.yml"), data, 0644)

	if err != nil {
		return err
	}

	data, err = genDockerfile(procfile)

	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(base, "Dockerfile"), data, 0644)

	if err != nil {
		return err
	}

	err = run("docker-compose", "up")

	if err != nil {
		return err
	}

	return nil
}

func genDockerCompose(procs map[string]string) ([]byte, error) {
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

func genDockerfile(procs map[string]string) ([]byte, error) {
	return []byte(`FROM convox/cedar`), nil
}

func parseProcfile(data []byte) (map[string]string, error) {
	pf := map[string]string{}

	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		parts := procfileEntryRegexp.FindStringSubmatch(scanner.Text())

		if len(parts) > 0 {
			pf[parts[1]] = parts[2]
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("Reading Procfile: %s", err)
	}

	return pf, nil
}
