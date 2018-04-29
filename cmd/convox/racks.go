package main

import (
	"fmt"

	"github.com/convox/rack/cmd/convox/stdcli"
	"gopkg.in/urfave/cli.v1"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "racks",
		Description: "list your Convox racks",
		Usage:       "",
		Action:      cmdRacks,
	})
}

func cmdRacks(c *cli.Context) error {
	if len(c.Args()) > 0 {
		return stdcli.Error(fmt.Errorf("`convox racks` does not take arguments. Perhaps you meant `convox rack`?"))
	}

	racks := rackList()

	t := stdcli.NewTable("RACK", "STATUS")

	for _, rack := range racks {
		t.AddRow(rack.Name, rack.Status)
	}

	t.Print()

	return nil
}
