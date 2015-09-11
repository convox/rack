package main

import (
	"fmt"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
	"github.com/dustin/go-humanize"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "ps",
		Description: "list an app's processes",
		Usage:       "",
		Action:      cmdPs,
		Flags:       []cli.Flag{appFlag},
		Subcommands: []cli.Command{
			{
				Name:        "stop",
				Description: "stop a process",
				Usage:       "<id>",
				Action:      cmdPsStop,
				Flags:       []cli.Flag{appFlag},
			},
		},
	})
}

func cmdPs(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	ps, err := rackClient(c).GetProcesses(app)

	if err != nil {
		stdcli.Error(err)
		return
	}

	t := stdcli.NewTable("ID", "NAME", "CPU", "MEM", "STARTED", "COMMAND")

	for _, p := range ps {
		t.AddRow(p.Id, p.Name, fmt.Sprintf("%0.2f%%", p.Cpu*100), fmt.Sprintf("%0.2f%%", p.Memory*100), humanize.Time(p.Started), p.Command)
	}

	t.Print()
}

func cmdPsStop(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	if len(c.Args()) != 1 {
		stdcli.Usage(c, "stop")
		return
	}

	id := c.Args()[0]

	fmt.Printf("Stopping %s... ", id)

	_, err = rackClient(c).StopProcess(app, id)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Println("OK")
}
