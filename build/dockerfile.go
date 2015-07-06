package build

import (
	"io/ioutil"
	"path/filepath"
)

func Dockerfile(base string, app string) error {
	err := run("build", base, "docker", "build", "-t", app, base)

	if err != nil {
		return err
	}

	data, err := query("docker", "inspect", "-f", "{{ json .ContainerConfig.ExposedPorts }}", app)

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
