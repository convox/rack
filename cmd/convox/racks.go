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
		return stdcli.ExitError(fmt.Errorf("`convox racks` does not take arguments. Perhaps you meant `convox racks`?"))
	}

	racks, err := rackClient(c).Racks()
	if err != nil {
		return stdcli.ExitError(err)
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
	return nil
}
