package main

import (
	"fmt"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/Godeps/_workspace/src/github.com/fatih/color"
	"github.com/convox/cli/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "system",
		Description: "[deprecated: use `rack`]",
		Usage:       "",
		Action:      cmdSystem,
		Subcommands: []cli.Command{
			{
				Name:        "update",
				Description: "update rack to the latest version",
				Usage:       "[version]",
				Action:      cmdSystemUpdate,
			},
			{
				Name:        "scale",
				Description: "scale the rack capacity",
				Usage:       "",
				Action:      cmdSystemScale,
				Flags: []cli.Flag{
					cli.IntFlag{
						Name:  "count",
						Usage: "horizontally scale the instance count, e.g. 3 or 10",
					},
					cli.StringFlag{
						Name:  "type",
						Usage: "vertically scale the instance type, e.g. t2.small or c3.xlargs",
					},
				},
			},
		},
	})
}

func cmdSystemWarn() {
	fmt.Println(color.YellowString("WARNING: `convox system` is deprecated; use `convox rack` instead."))
}

func cmdSystem(c *cli.Context) {
	cmdSystemWarn()
	cmdRack(c)
}

func cmdSystemUpdate(c *cli.Context) {
	cmdSystemWarn()
	cmdRackUpdate(c)
}

func cmdSystemScale(c *cli.Context) {
	cmdSystemWarn()
	cmdRackScale(c)
}
