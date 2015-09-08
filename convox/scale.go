package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "scale",
		Description: "scale an app's processes",
		Usage:       "PROCESS [--count 2] [--memory 512]",
		Action:      cmdScale,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "app",
				Usage: "App name. Inferred from current directory if not specified.",
			},
			cli.IntFlag{
				Name:  "count",
				Value: 0,
				Usage: "Number of processes to keep running for specified process type.",
			},
			cli.IntFlag{
				Name:  "memory",
				Value: 0,
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

	if len(c.Args()) == 0 {
		displayFormation(app)
		return
	}

	process := c.Args()[0]
	count := c.Int("count")
	memory := c.Int("memory")

	err = rackClient().SetFormation(app, process, count, memory)

	if err != nil {
		stdcli.Error(err)
		return
	}

	displayFormation(app)
}

func displayFormation(app string) {
	formation, err := rackClient().ListFormation(app)

	if err != nil {
		stdcli.Error(err)
		return
	}

	t := stdcli.NewTable("NAME", "COUNT", "MEMORY", "PORTS")

	for _, f := range formation {
		ports := []string{}

		for _, p := range f.Ports {
			ports = append(ports, strconv.Itoa(p))
		}

		t.AddRow(f.Name, fmt.Sprintf("%d", f.Count), fmt.Sprintf("%d", f.Memory), strings.Join(ports, " "))
	}

	t.Print()

}
