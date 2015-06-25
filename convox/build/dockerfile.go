package build

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
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

	data, err = ManifestFromInspect(data)

	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(base, "docker-compose.yml"), data, 0644)

	if err != nil {
		return err
	}

	return nil
}
