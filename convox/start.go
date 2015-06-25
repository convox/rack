package main

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "start",
		Description: "start an app for local development",
		Usage:       "<directory>",
		Action:      cmdStart,
	})
}

func cmdStart(c *cli.Context) {
	base := "."

	if len(c.Args()) > 0 {
		base = c.Args()[0]
	}

	base, err := filepath.Abs(base)

	if err != nil {
		panic(err)
	}

	cmdBuild(c)

	err = run("docker-compose", "up")

	if err != nil {
		panic(err)
	}
}

func run(bin string, args ...string) error {
	cmd := exec.Command(bin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
