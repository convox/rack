package build

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/convox/cli/stdcli"
)

func Dockerfile(base string) error {
	app := filepath.Base(base)
	image := fmt.Sprintf("%s-app", app)

	err := stdcli.Run("docker", "build", "-t", image, base)

	if err != nil {
		return err
	}

	data, err := stdcli.Query("docker", "inspect", "-f", "{{ json .ContainerConfig.ExposedPorts }}", image)

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
