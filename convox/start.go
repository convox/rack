package main

import (
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

	err = stdcli.Run("docker-compose", "up")

	if err != nil {
		panic(err)
	}
}
