package main

import (
	"fmt"

	"/github.com/codegangsta/cli"
	"github.com/convox/rack/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "switch",
		Description: "switch to another Convox rack",
		Usage:       "[hostname]",
		Action:      cmdSwitch,
	})
}

func cmdSwitch(c *cli.Context) {
	if len(c.Args()) < 1 {
		stdcli.Usage(c, "switch")
		return
	}

	host := c.Args()[0]

	err := switchHost(host)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Printf("Switched to %s\n", host)
}
