package main

import (
	// "os"
	// "path/filepath"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	//"github.com/convox/cli/convox/start"
	"github.com/convox/cli/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "env",
		Description: "manage an app's environment variables",
		Usage:       "set|change|delete",
		Subcommands: []cli.Command{
			{
				Name:   "set",
				Usage:  "VARIABLE=VALUE",
				Action: nil,
			},
			{
				Name:   "change",
				Usage:  "VARIABLE=VALUE",
				Action: nil,
			},
			{
				Name:   "delete",
				Usage:  "VARIABLE",
				Action: nil,
			},
		},
	})
}
