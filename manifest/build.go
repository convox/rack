package manifest

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/convox/cli/Godeps/_workspace/src/gopkg.in/yaml.v2"
)

func buildDockerCompose(dir string) (*Manifest, error) {
	data, err := ioutil.ReadFile(filepath.Join(dir, "docker-compose.yml"))

	if err != nil {
		return nil, err
	}

	var m Manifest

	err = yaml.Unmarshal(data, &m)

	if err != nil {
		return nil, err
	}

	if denv := filepath.Join(dir, ".env"); exists(denv) {
		data, err := ioutil.ReadFile(denv)

		if err != nil {
			return nil, err
		}

		scanner := bufio.NewScanner(bytes.NewReader(data))

		for scanner.Scan() {
			if strings.Index(scanner.Text(), "=") > -1 {
				parts := strings.SplitN(scanner.Text(), "=", 2)

				err := os.Setenv(parts[0], parts[1])

				if err != nil {
					return nil, err
				}
			}
		}

		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}

	for name, entry := range m {
		for i, volume := range entry.Volumes {
			parts := strings.Split(volume, ":")

			for j, part := range parts {
				if !filepath.IsAbs(part) {
					parts[j] = filepath.Join(dir, part)
				}
			}

			m[name].Volumes[i] = strings.Join(parts, ":")
		}
	}

	return &m, nil
}

var exposeEntryRegexp = regexp.MustCompile(`^EXPOSE\s+(\d+)`)

func buildDockerfile(dir string) (*Manifest, error) {
	entry := ManifestEntry{
		Build: ".",
	}

	data, err := ioutil.ReadFile(filepath.Join(dir, "Dockerfile"))

	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))

	current := 5000

	for scanner.Scan() {
		parts := exposeEntryRegexp.FindStringSubmatch(scanner.Text())

		if len(parts) > 1 {
			entry.Ports = append(entry.Ports, fmt.Sprintf("%d:%s", current, strings.Split(parts[1], "/")[0]))
			current += 100
		}
	}

	manifest := &Manifest{"main": entry}

	err = manifest.Write(filepath.Join(dir, "docker-compose.yml"))

	if err != nil {
		return nil, err
	}

	return manifest, nil
}

var procfileEntryRegexp = regexp.MustCompile("^([A-Za-z0-9_]+):\\s*(.+)$")

func buildProcfile(dir string) (*Manifest, error) {
	m := Manifest{}

	err := injectDockerfile(dir)

	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadFile(filepath.Join(dir, "Procfile"))

	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))

	current := 5000

	for scanner.Scan() {
		parts := procfileEntryRegexp.FindStringSubmatch(scanner.Text())

		if len(parts) > 0 {
			m[parts[1]] = ManifestEntry{
				Build:   ".",
				Command: parts[2],
				Ports:   []string{fmt.Sprintf("%d:3000", current)},
			}

			current += 100
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	err = m.Write(filepath.Join(dir, "docker-compose.yml"))

	if err != nil {
		return nil, err
	}

	return &m, err
}
