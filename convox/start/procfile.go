package start

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	yaml "github.com/convox/cli/Godeps/_workspace/src/gopkg.in/yaml.v2"
)

type Manifest map[string]ManifestEntry

type ManifestEntry struct {
	Build       string      `yaml:"build"`
	Command     interface{} `yaml:"command"`
	Environment []string    `yaml:"environment"`
	Ports       []string    `yaml:"ports"`
}

var procfileEntryRegexp = regexp.MustCompile("^([A-Za-z0-9_]+):\\s*(.+)$")

func Procfile(base string) error {
	app := filepath.Base(base)
	image := fmt.Sprintf("%s-app", app)

	dockerfile := filepath.Join(base, "Dockerfile")

	err := ioutil.WriteFile(dockerfile, []byte("FROM convox/cedar"), 0644)

	if err != nil {
		return err
	}

	err = run("docker", "build", "-f", dockerfile, "-t", image, base)

	if err != nil {
		return err
	}

	err = os.Remove(dockerfile)

	if err != nil {
		return err
	}

	data, err := query("docker", "inspect", "-f", "{{ json .ContainerConfig.ExposedPorts }}", image)

	if err != nil {
		return err
	}

	var ports map[string]interface{}

	err = json.Unmarshal(data, &ports)

	if err != nil {
		return err
	}

	args := []string{"run"}
	cur := 5000

	for port, _ := range ports {
		args = append(args, "-p")
		args = append(args, fmt.Sprintf("%d:%s", cur, strings.Split(port, "/")[0]))
		cur += 1
	}

	args = append(args, image)

	data, err = ioutil.ReadFile(filepath.Join(base, "Procfile"))

	if err != nil {
		return err
	}

	procfile, err := parseProcfile(data)

	if err != nil {
		return err
	}

	args = append(args, procfile["web"])

	err = run("docker", args...)

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
