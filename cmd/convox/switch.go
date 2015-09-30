package main

import (
	"fmt"

	"github.com/codegangsta/cli"
	"github.com/convox/rack/cmd/convox/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "switch",
		Description: "switch to another Convox rack",
		Usage:       "[rack name]",
		Action:      cmdSwitch,
	})
}

func cmdSwitch(c *cli.Context) {
	if len(c.Args()) < 1 {
		stdcli.Usage(c, "switch")
		return
	}

	rackName := c.Args()[0]

	res, err := rackClient(c).Switch(rackName)

	if err != nil {
		cmdLogin(c)
		return
	}

	switch {
	case res["source"] == "rack":
		cmdLogin(c)
	case res["source"] == "grid":
		fmt.Println(res["message"])
	}
}
