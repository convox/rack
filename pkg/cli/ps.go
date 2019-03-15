package cli

import (
	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/sdk"
	"github.com/convox/stdcli"
)

func init() {
	register("ps", "list app processes", Ps, stdcli.CommandOptions{
		Flags:    append(stdcli.OptionFlags(structs.ProcessListOptions{}), flagApp, flagRack),
		Validate: stdcli.Args(0),
	})

	register("ps info", "get information about a process", PsInfo, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagApp, flagRack},
		Validate: stdcli.Args(1),
	})

	register("ps stop", "stop a process", PsStop, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagApp, flagRack},
		Validate: stdcli.Args(1),
	})
}

func Ps(rack sdk.Interface, c *stdcli.Context) error {
	var opts structs.ProcessListOptions

	if err := c.Options(&opts); err != nil {
		return err
	}

	ps, err := rack.ProcessList(app(c), opts)
	if err != nil {
		return err
	}

	t := c.Table("ID", "SERVICE", "STATUS", "RELEASE", "STARTED", "COMMAND")

	for _, p := range ps {
		t.AddRow(p.Id, p.Name, p.Status, p.Release, helpers.Ago(p.Started), p.Command)
	}

	return t.Print()
}

func PsInfo(rack sdk.Interface, c *stdcli.Context) error {
	i := c.Info()

	ps, err := rack.ProcessGet(app(c), c.Arg(0))
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
	i.Add("Status", ps.Status)

	return i.Print()
}

func PsStop(rack sdk.Interface, c *stdcli.Context) error {
	c.Startf("Stopping <process>%s</process>", c.Arg(0))

	if err := rack.ProcessStop(app(c), c.Arg(0)); err != nil {
		return err
	}

	return c.OK()
}
