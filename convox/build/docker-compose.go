package build

import (
	"github.com/convox/cli/stdcli"
)

func DockerCompose(base string) error {
	err := stdcli.Run("docker-compose", "build")

	if err != nil {
		return err
	}

	return nil
}
