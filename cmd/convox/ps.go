package main

import (
	"fmt"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/rack/client"
	"github.com/convox/rack/cmd/convox/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "ps",
		Description: "list an app's processes",
		Usage:       "",
		Action:      cmdPs,
		Flags: []cli.Flag{
			appFlag,
			cli.BoolFlag{
				Name:  "stats",
				Usage: "display process cpu/memory stats",
			},
		},
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

	ps, err := rackClient(c).GetProcesses(app, c.Bool("stats"))

	if err != nil {
		stdcli.Error(err)
		return
	}

	if c.Bool("stats") {
		t := stdcli.NewTable("ID", "NAME", "RELEASE", "CPU", "MEM", "STARTED", "COMMAND")

		for _, p := range ps {
			t.AddRow(prettyId(p), p.Name, p.Release, fmt.Sprintf("%0.2f%%", p.Cpu*100), fmt.Sprintf("%0.2f%%", p.Memory*100), humanizeTime(p.Started), p.Command)
		}

		t.Print()
	} else {
		t := stdcli.NewTable("ID", "NAME", "RELEASE", "STARTED", "COMMAND")

		for _, p := range ps {
			t.AddRow(prettyId(p), p.Name, p.Release, humanizeTime(p.Started), p.Command)
		}

		t.Print()
	}
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

func prettyId(p client.Process) string {
	if p.Id == "pending" {
		return "[PENDING]"
	}

	return p.Id
}
