package main

import (
	"os"
	"path/filepath"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/convox/start"
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

	switch {
	case exists(filepath.Join(base, "Dockerfile")):
		err = start.Dockerfile(base)
	case exists(filepath.Join(base, "Procfile")):
		err = start.Procfile(base)
	}

	if err != nil {
		panic(err)
	}
}

func exists(filename string) bool {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false
	}

	return true
}
