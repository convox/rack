package build

import (
	"path/filepath"
	"strings"

	"github.com/convox/cli/stdcli"
)

func DockerCompose(dir string) error {
	err := stdcli.Run("docker-compose", "build")

	if err != nil {
		return err
	}

	m, err := ManifestFromPath(filepath.Join(dir, "docker-compose.yml"))

	if err != nil {
		return err
	}

	proj := strings.Replace(filepath.Base(dir), "-", "", -1)

	images := m.ImageNames(proj)
	for i := 0; i < len(images); i++ {
		err = stdcli.Run("docker", "pull", images[i])

		if err != nil {
			return err
		}
	}

	return nil
}
