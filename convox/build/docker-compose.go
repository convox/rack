package build

import "github.com/convox/cli/stdcli"

func DockerCompose(dir string, app string) error {
	err := stdcli.Run("docker-compose", "-p", app, "build")

	if err != nil {
		return err
	}

	err = stdcli.Run("docker-compose", "-p", app, "pull")

	if err != nil {
		return err
	}

	return nil
}
