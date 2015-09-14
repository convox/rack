package main

import (
	"github.com/convox/rack/cmd/convox/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/rack/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "build",
		Description: "create a new build",
		Usage:       "",
		Action:      cmdBuildsCreate,
		Flags:       []cli.Flag{appFlag},
	})
}
