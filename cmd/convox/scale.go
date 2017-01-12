package main

import (
	"fmt"

	"github.com/convox/rack/client"
	"github.com/convox/rack/cmd/convox/stdcli"
	"gopkg.in/urfave/cli.v1"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "scale",
		Description: "scale an app's processes",
		Usage:       "<process> [--count=2] [--memory=256] [--cpu=256]",
		Action:      cmdScale,
		Flags: []cli.Flag{
			appFlag,
			rackFlag,
			cli.IntFlag{
				Name:  "count",
				Usage: "Number of processes to keep running for specified process type.",
			},
			cli.IntFlag{
				Name:  "memory",
				Usage: "Amount of memory, in MB, available to specified process type.",
			},
			cli.IntFlag{
				Name:  "cpu",
				Usage: "CPU units available to specified process type.",
			},
			cli.BoolFlag{
				Name:   "wait",
				EnvVar: "CONVOX_WAIT",
				Usage:  "wait for app to finish scaling before returning",
			},
		},
	})
}

func cmdScale(c *cli.Context) error {
	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.Error(err)
	}

	opts := client.FormationOptions{}

	if c.IsSet("count") {
		opts.Count = c.String("count")
	}

	if c.IsSet("cpu") {
		opts.CPU = c.String("cpu")
	}

	if c.IsSet("memory") {
		opts.Memory = c.String("memory")
	}

	// validate single process type argument
	switch len(c.Args()) {
	case 0:
		if opts.Memory != "" || opts.CPU != "" || opts.Count != "" {
			return stdcli.Error(fmt.Errorf("missing process name"))
		}

		displayFormation(c, app)
		return nil
	case 1:
		if opts.Count == "" && opts.CPU == "" && opts.Memory == "" {
			displayFormation(c, app)
			return nil
		}
		// fall through to scale API call
	default:
		stdcli.Usage(c, "scale")
		return nil
	}

	process := c.Args()[0]

	err = rackClient(c).SetFormation(app, process, opts)
	if err != nil {
		return stdcli.Error(err)
	}

	err = displayFormation(c, app)
	if err != nil {
		return stdcli.Error(err)
	}

	if c.Bool("wait") {
		fmt.Printf("Waiting for stabilization... ")

		a, err := rackClient(c).GetApp(app)
		if err != nil {
			return stdcli.Error(err)
		}

		if err := waitForReleasePromotion(c, app, a.Release); err != nil {
			return stdcli.Error(err)
		}

		fmt.Println("OK")
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

	t := stdcli.NewTable("NAME", "DESIRED", "RUNNING", "CPU", "MEMORY")

	for _, f := range formation {
		t.AddRow(f.Name, fmt.Sprintf("%d", f.Count), fmt.Sprintf("%d", running[f.Name]), fmt.Sprintf("%d", f.CPU), fmt.Sprintf("%d", f.Memory))
	}

	t.Print()
	return nil
}
