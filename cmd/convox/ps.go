package main

import (
	"fmt"
	"strconv"

	"github.com/convox/rack/client"
	"github.com/convox/rack/cmd/convox/helpers"
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
			rackFlag,
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
				Flags:       []cli.Flag{appFlag, rackFlag},
			},
			{
				Name:        "stop",
				Description: "stop a process",
				Usage:       "<id>",
				Action:      cmdPsStop,
				Flags:       []cli.Flag{appFlag, rackFlag},
			},
		},
	})
}

func cmdPs(c *cli.Context) error {
	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.Error(err)
	}

	ps, err := rackClient(c).GetProcesses(app, c.Bool("stats"))
	if err != nil {
		return stdcli.Error(err)
	}

	if c.Bool("stats") {
		fm, err := rackClient(c).ListFormation(app)
		if err != nil {
			return stdcli.Error(err)
		}

		system, err := rackClient(c).GetSystem()
		if err != nil {
			return stdcli.Error(err)
		}

		params, err := rackClient(c).ListParameters(system.Name)
		if err != nil {
			return stdcli.Error(err)
		}

		memory, err := strconv.Atoi(params["BuildMemory"])
		if err != nil {
			return stdcli.Error(err)
		}

		fm = append(fm, client.FormationEntry{
			Name:   "build",
			Memory: memory,
		})

		displayProcessesStats(ps, fm, false)

		return nil
	}

	displayProcesses(ps, false)

	return nil
}

func displayProcesses(ps []client.Process, showApp bool) {
	var t *stdcli.Table
	if showApp {
		t = stdcli.NewTable("ID", "APP", "GROUP", "NAME", "RELEASE", "STARTED", "COMMAND")
	} else {
		t = stdcli.NewTable("ID", "GROUP", "NAME", "RELEASE", "STARTED", "COMMAND")
	}

	for _, p := range ps {
		if showApp {
			t.AddRow(prettyId(p), p.Group, p.App, p.Name, p.Release, helpers.HumanizeTime(p.Started), p.Command)
		} else {
			t.AddRow(prettyId(p), p.Group, p.Name, p.Release, helpers.HumanizeTime(p.Started), p.Command)
		}
	}

	t.Print()
}

func displayProcessesStats(ps []client.Process, fm client.Formation, showApp bool) {
	var t *stdcli.Table
	if showApp {
		t = stdcli.NewTable("ID", "NAME", "APP", "RELEASE", "CPU %", "MEM", "MEM %", "STARTED", "COMMAND")
	} else {
		t = stdcli.NewTable("ID", "NAME", "RELEASE", "CPU %", "MEM", "MEM %", "STARTED", "COMMAND")
	}

	for _, p := range ps {
		for _, f := range fm {
			if f.Name != p.Name {
				continue
			}
			if showApp {
				t.AddRow(prettyId(p), p.Name, p.App, p.Release, fmt.Sprintf("%0.2f%%", p.Cpu), fmt.Sprintf("%0.1fMB/%dMB", p.Memory*float64(f.Memory), f.Memory), fmt.Sprintf("%0.2f%%", p.Memory*100), helpers.HumanizeTime(p.Started), p.Command)
			} else {
				t.AddRow(prettyId(p), p.Name, p.Release, fmt.Sprintf("%0.2f%%", p.Cpu), fmt.Sprintf("%0.1fMB/%dMB", p.Memory*float64(f.Memory), f.Memory), fmt.Sprintf("%0.2f%%", p.Memory*100), helpers.HumanizeTime(p.Started), p.Command)
			}
		}
	}

	t.Print()
}

func cmdPsInfo(c *cli.Context) error {
	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.Error(err)
	}

	if len(c.Args()) != 1 {
		stdcli.Usage(c, "info")
		return nil
	}

	id := c.Args()[0]

	p, err := rackClient(c).GetProcess(app, id)
	if err != nil {
		return stdcli.Error(err)
	}

	fmt.Printf("Id       %s\n", p.Id)
	fmt.Printf("Name     %s\n", p.Name)
	fmt.Printf("Release  %s\n", p.Release)
	fmt.Printf("CPU      %0.2f%%\n", p.Cpu)
	fmt.Printf("Memory   %0.2f%%\n", p.Memory*100)
	fmt.Printf("Started  %s\n", helpers.HumanizeTime(p.Started))
	fmt.Printf("Command  %s\n", p.Command)

	return nil
}

func cmdPsStop(c *cli.Context) error {
	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.Error(err)
	}

	if len(c.Args()) != 1 {
		stdcli.Usage(c, "stop")
		return nil
	}

	id := c.Args()[0]

	fmt.Printf("Stopping %s... ", id)

	_, err = rackClient(c).StopProcess(app, id)
	if err != nil {
		return stdcli.Error(err)
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
