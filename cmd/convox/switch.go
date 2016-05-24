package main

import (
	"fmt"

	"github.com/convox/rack/cmd/convox/stdcli"
	"gopkg.in/urfave/cli.v1"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "switch",
		Description: "switch to another Convox rack",
		Usage:       "[rack name]",
		Action:      cmdSwitch,
	})
}

func cmdSwitch(c *cli.Context) error {
	if len(c.Args()) < 1 {
		stdcli.Usage(c, "switch")
		return nil
	}

	rackName := c.Args()[0]

	res, err := rackClient(c).Switch(rackName)
	if err != nil {
		cmdLogin(c)
		return nil
	}

	switch {
	case res["source"] == "rack":
		cmdLogin(c)
	case res["source"] == "grid":
		fmt.Println(res["message"])
	}
	return nil
}
