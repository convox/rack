package main

import (
	"fmt"

	"gopkg.in/urfave/cli.v1"
	"github.com/convox/rack/cmd/convox/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "racks",
		Description: "list your Convox racks",
		Usage:       "",
		Action:      cmdRacks,
	})
}

func cmdRacks(c *cli.Context) {
	if len(c.Args()) > 0 {
		stdcli.Error(fmt.Errorf("`convox racks` does not take arguments. Perhaps you meant `convox racks`?"))
		return
	}

	racks, err := rackClient(c).Racks()
	if err != nil {
		stdcli.Error(err)
		return
	}

	t := stdcli.NewTable("RACK", "STATUS")
	for _, rack := range racks {
		name := rack.Name
		if rack.Organization != nil {
			name = fmt.Sprintf("%s/%s", rack.Organization.Name, name)
		}
		t.AddRow(name, rack.Status)
	}
	t.Print()
}
