package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/convox/build"
	"github.com/convox/cli/stdcli"
)

func Build(dir string, app string) error {
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
		err = build.DockerCompose(dir, app)
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
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "app",
				Usage: "app name. Inferred from current directory if not specified.",
			},
		},
	})
}

func cmdBuild(c *cli.Context) {
	wd := "."

	if len(c.Args()) > 0 {
		wd = c.Args()[0]
	}

	dir, app, err := stdcli.DirApp(c, wd)

	if err != nil {
		stdcli.Error(err)
		return
	}

	err = Build(dir, app)

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
