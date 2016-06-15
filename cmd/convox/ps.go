package main

import (
	"fmt"

	"github.com/convox/rack/client"
	"github.com/convox/rack/cmd/convox/stdcli"
	"gopkg.in/urfave/cli.v1"
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
				Name:        "info",
				Description: "show info for a process",
				Usage:       "<id>",
				Action:      cmdPsInfo,
				Flags:       []cli.Flag{appFlag},
			},
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

func cmdPs(c *cli.Context) error {
	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.ExitError(err)
	}

	ps, err := rackClient(c).GetProcesses(app, c.Bool("stats"))
	if err != nil {
		return stdcli.ExitError(err)
	}

	if c.Bool("stats") {
		t := stdcli.NewTable("ID", "NAME", "RELEASE", "CPU", "SIZE", "CPU %", "MEM %", "STARTED", "COMMAND")

		for _, p := range ps {
			t.AddRow(prettyId(p), p.Name, p.Release, fmt.Sprintf("%d", p.Cpu), fmt.Sprintf("%d", p.Size), fmt.Sprintf("%0.2f%%", p.Cpu), fmt.Sprintf("%0.2f%%", p.Memory*100), humanizeTime(p.Started), p.Command)
		}

		t.Print()
	} else {
		t := stdcli.NewTable("ID", "NAME", "RELEASE", "CPU", "SIZE", "STARTED", "COMMAND")

		for _, p := range ps {
			t.AddRow(prettyId(p), p.Name, p.Release, fmt.Sprintf("%d", p.Cpu), fmt.Sprintf("%d", p.Size), humanizeTime(p.Started), p.Command)
		}

		t.Print()
	}

	return nil
}

func cmdPsInfo(c *cli.Context) error {
	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.ExitError(err)
	}

	if len(c.Args()) != 1 {
		stdcli.Usage(c, "info")
		return nil
	}

	id := c.Args()[0]

	p, err := rackClient(c).GetProcess(app, id)

	if err != nil {
		return stdcli.ExitError(err)
	}

	fmt.Printf("Id       %s\n", p.Id)
	fmt.Printf("Name     %s\n", p.Name)
	fmt.Printf("Release  %s\n", p.Release)
	fmt.Printf("Size     %d\n", p.Size)
	fmt.Printf("CPU      %0.2f%%\n", p.Cpu)
	fmt.Printf("Memory   %0.2f%%\n", p.Memory*100)
	fmt.Printf("Started  %s\n", humanizeTime(p.Started))
	fmt.Printf("Command  %s\n", p.Command)

	return nil
}

func cmdPsStop(c *cli.Context) error {
	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.ExitError(err)
	}

	if len(c.Args()) != 1 {
		stdcli.Usage(c, "stop")
		return nil
	}

	id := c.Args()[0]

	fmt.Printf("Stopping %s... ", id)

	_, err = rackClient(c).StopProcess(app, id)
	if err != nil {
		return stdcli.ExitError(err)
	}

	fmt.Println("OK")
	return nil
}

func prettyId(p client.Process) string {
	if p.Id == "pending" {
		return "[PENDING]"
	}

	return p.Id
}
