package main

import (
	"github.com/convox/rack/helpers"
	"github.com/convox/rack/structs"
	"github.com/convox/stdcli"
)

func init() {
	CLI.Command("ps", "list app processes", Ps, stdcli.CommandOptions{
		Flags:    append(stdcli.OptionFlags(structs.ProcessListOptions{}), flagApp, flagRack),
		Validate: stdcli.Args(0),
	})
}

func Ps(c *stdcli.Context) error {
	var opts structs.ProcessListOptions

	if err := c.Options(&opts); err != nil {
		return err
	}

	ps, err := provider(c).ProcessList(app(c), opts)
	if err != nil {
		return err
	}

	t := c.Table("ID", "NAME", "RELEASE", "STARTED", "COMMAND")

	for _, p := range ps {
		t.AddRow(p.Id, p.Name, p.Release, helpers.Ago(p.Started), p.Command)
	}

	return t.Print()
}
