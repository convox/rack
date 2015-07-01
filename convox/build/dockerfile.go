package build

import (
	"io/ioutil"
	"path/filepath"

	"github.com/convox/cli/stdcli"
)

func Dockerfile(base string, app string) error {
	err := stdcli.Run("docker", "build", "-t", app, base)

	if err != nil {
		return err
	}

	data, err := stdcli.Query("docker", "inspect", "-f", "{{ json .ContainerConfig.ExposedPorts }}", app)

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
