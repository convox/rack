package build

import (
	"io/ioutil"
	"path/filepath"

	"github.com/convox/cli/stdcli"
)

func Default(base string, app string) error {
	err := ioutil.WriteFile(filepath.Join(base, "Dockerfile"), []byte(`FROM convox/cedar`), 0644)

	if err != nil {
		return err
	}

	err = stdcli.Run("docker", "build", "-t", app, base)

	if err != nil {
		return err
	}

	data, err := stdcli.Query("docker", "run", app, "cat /app/Procfile")

	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(base, "Procfile"), data, 0644)

	if err != nil {
		return err
	}

	procfile, err := parseProcfile(data)

	if err != nil {
		return err
	}

	data, err = ManifestFromProcfile(procfile)

	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(base, "docker-compose.yml"), data, 0644)

	if err != nil {
		return err
	}

	return nil
}
