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

	CLI.Command("ps info", "get information about a process", PsInfo, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagApp, flagRack},
		Validate: stdcli.Args(1),
	})

	CLI.Command("ps stop", "stop a process", PsStop, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagApp, flagRack},
		Validate: stdcli.Args(1),
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

func PsInfo(c *stdcli.Context) error {
	i := c.Info()

	ps, err := provider(c).ProcessGet(app(c), c.Arg(0))
	if err != nil {
		return err
	}

	i.Add("Id", ps.Id)
	i.Add("App", ps.App)
	i.Add("Command", ps.Command)
	i.Add("Instance", ps.Instance)
	i.Add("Release", ps.Release)
	i.Add("Service", ps.Name)
	i.Add("Started", helpers.Ago(ps.Started))

	return i.Print()
}

func PsStop(c *stdcli.Context) error {
	c.Startf("Stopping <process>%s</process>", c.Arg(0))

	if err := provider(c).ProcessStop(app(c), c.Arg(0)); err != nil {
		return err
	}

	return c.OK()
}
