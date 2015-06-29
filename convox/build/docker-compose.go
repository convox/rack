package build

import (
	"path/filepath"

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

	for key := range m {
		ps := m[key]

		if ps.Image != "" {
			err = stdcli.Run("docker", "pull", ps.Image)

			if err != nil {
				return err
			}
		}
	}

	return nil
}
