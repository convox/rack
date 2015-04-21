package start

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
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
		cur += 100
	}

	args = append(args, image)

	err = run("docker", args...)

	if err != nil {
		return err
	}

	return nil
}
