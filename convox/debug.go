package main

import (
	"fmt"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "debug",
		Description: "get an app's system events for debugging purposes",
		Usage:       "",
		Action:      cmdDebug,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "name",
				Usage: "app name. Inferred from current directory if not specified.",
			},
		},
	})
}

func cmdDebug(c *cli.Context) {
	name := c.String("name")

	if name == "" {
		name = DirAppName()
	}

	data, err := ConvoxGet(fmt.Sprintf("/apps/%s/events", name))

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Printf("%v\n", string(data))
}
