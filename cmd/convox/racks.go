package main

import (
	"github.com/convox/rack/Godeps/_workspace/src/github.com/codegangsta/cli"
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
	racks, err := rackClient(c).Racks()
	if err != nil {
		stdcli.Error(err)
		return
	}

	t := stdcli.NewTable("RACK", "STATUS")
	for _, rack := range racks {
		t.AddRow(rack.Name, rack.Status)
	}
	t.Print()
}
