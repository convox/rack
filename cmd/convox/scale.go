package main

import (
	"fmt"

	"github.com/convox/rack/cmd/convox/stdcli"
	"gopkg.in/urfave/cli.v1"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "scale",
		Description: "scale an app's processes",
		Usage:       "<process> [--count=2] [--memory=512]",
		Action:      cmdScale,
		Flags: []cli.Flag{appFlag,
			cli.StringFlag{
				Name:  "count",
				Value: "",
				Usage: "Number of processes to keep running for specified process type.",
			},
			cli.StringFlag{
				Name:  "memory",
				Value: "",
				Usage: "Amount of memory, in MB, available to specified process type.",
			},
		},
	})
}

func cmdScale(c *cli.Context) error {
	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.ExitError(err)
	}

	count := c.String("count")
	memory := c.String("memory")

	if len(c.Args()) == 0 && count == "" && memory == "" {
		err = displayFormation(c, app)
		if err != nil {
			return stdcli.ExitError(err)
		}
		return nil
	}

	if len(c.Args()) != 1 || (count == "" && memory == "") {
		stdcli.Usage(c, "scale")
		return nil
	}

	process := c.Args()[0]

	err = rackClient(c).SetFormation(app, process, count, memory)
	if err != nil {
		return stdcli.ExitError(err)
	}

	err = displayFormation(c, app)
	if err != nil {
		return stdcli.ExitError(err)
	}
	return nil
}

func displayFormation(c *cli.Context, app string) error {
	formation, err := rackClient(c).ListFormation(app)
	if err != nil {
		return err
	}

	pss, err := rackClient(c).GetProcesses(app, false)
	if err != nil {
		return err
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
	return nil
}
