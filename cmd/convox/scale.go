package main

import (
	"fmt"

	"github.com/codegangsta/cli"
	"github.com/convox/rack/cmd/convox/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "scale",
		Description: "scale an app's processes",
		Usage:       "<process> [--count=2] [--memory=512]",
		Action:      cmdScale,
		Flags: []cli.Flag{appFlag,
			cli.IntFlag{
				Name:  "count",
				Usage: "Number of processes to keep running for specified process type.",
			},
			cli.IntFlag{
				Name:  "memory",
				Usage: "Amount of memory, in MB, available to specified process type.",
			},
		},
	})
}

func cmdScale(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		stdcli.Error(err)
		return
	}

	// initialize to invalid values that indicate no change
	count := -1
	memory := -1

	if c.IsSet("count") {
		count = c.Int("count")
	}

	if c.IsSet("memory") {
		memory = c.Int("memory")
	}

	// validate single process type argument
	switch len(c.Args()) {
	case 0:
		displayFormation(c, app)
		return
	case 1:
		if count == -1 && memory == -1 {
			displayFormation(c, app)
			return
		}
		// fall through to scale API call
	default:
		stdcli.Usage(c, "scale")
		return
	}

	process := c.Args()[0]

	err = rackClient(c).SetFormation(app, process, count, memory)
	if err != nil {
		stdcli.Error(err)
		return
	}

	displayFormation(c, app)
}

func displayFormation(c *cli.Context, app string) {
	formation, err := rackClient(c).ListFormation(app)
	if err != nil {
		stdcli.Error(err)
		return
	}

	pss, err := rackClient(c).GetProcesses(app, false)
	if err != nil {
		stdcli.Error(err)
		return
	}

	running := map[string]int{}

	for _, ps := range pss {
		if ps.Id != "pending" {
			running[ps.Name] += 1
		}
	}

	t := stdcli.NewTable("NAME", "DESIRED", "RUNNING", "MEMORY")

	for _, f := range formation {
		t.AddRow(f.Name, fmt.Sprintf("%d", f.Count), fmt.Sprintf("%d", running[f.Name]), fmt.Sprintf("%d", f.Memory))
	}

	t.Print()
}
