package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/convox/build"
	"github.com/convox/cli/stdcli"
)

func Build(dir string) error {
	dir, err := filepath.Abs(dir)

	if err != nil {
		panic(err)
	}

	err = os.Chdir(dir)

	if err != nil {
		return err
	}

	switch {
	case exists(filepath.Join(dir, "docker-compose.yml")):
		fmt.Printf("Docker Compose app detected.\n")
		err = build.DockerCompose(dir)
	case exists(filepath.Join(dir, "Dockerfile")):
		fmt.Printf("Dockerfile app detected. Writing docker-compose.yml.\n")
		err = build.Dockerfile(dir)
	case exists(filepath.Join(dir, "Procfile")):
		fmt.Printf("Procfile app detected. Writing Dockerfile and docker-compose.yml.\n")
		err = build.Procfile(dir)
	default:
		fmt.Printf("Nothing detected. Writing Procfile, Dockerfile and docker-compose.yml.\n")
		err = build.Default(dir)
	}

	if err != nil {
		return err
	}

	return nil
}

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "build",
		Description: "build an app for local development",
		Usage:       "<directory>",
		Action:      cmdBuild,
	})
}

func cmdBuild(c *cli.Context) {
	base := "."

	if len(c.Args()) > 0 {
		base = c.Args()[0]
	}

	err := Build(base)

	if err != nil {
		stdcli.Error(err)
		return
	}

}

func exists(filename string) bool {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false
	}

	return true
}
