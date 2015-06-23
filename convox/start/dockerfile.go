package start

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"

	yaml "github.com/convox/cli/Godeps/_workspace/src/gopkg.in/yaml.v2"
)

func Dockerfile(base string) error {
	app := filepath.Base(base)
	image := fmt.Sprintf("%s-app", app)

	err := run("docker", "build", "-t", image, base)

	if err != nil {
		return err
	}

	data, err := query("docker", "inspect", "-f", "{{ json .ContainerConfig.ExposedPorts }}", image)

	if err != nil {
		return err
	}

	data, err = genInspectDockerCompose(data)

	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(base, "docker-compose.yml"), data, 0644)

	if err != nil {
		return err
	}

	err = run("docker-compose", "up")

	if err != nil {
		return err
	}

	return nil
}

func genInspectDockerCompose(data []byte) ([]byte, error) {
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
