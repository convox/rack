package build

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
)

func Default(base string) error {
	app := filepath.Base(base)
	image := fmt.Sprintf("%s-app", app)

	err := ioutil.WriteFile(filepath.Join(base, "Dockerfile"), []byte(`FROM convox/cedar`), 0644)

	if err != nil {
		return err
	}

	err = run("docker", "build", "-t", image, base)

	if err != nil {
		return err
	}

	data, err := query("docker", "run", image, "cat /app/Procfile")

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

	err = run("docker-compose", "up")

	if err != nil {
		return err
	}

	return nil
}
