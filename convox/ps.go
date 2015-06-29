package main

import (
	"encoding/json"
	"fmt"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
)

type Process struct {
	Name    string
	Command string
	Count   int

	ServiceType string

	App string
}

type Processes []Process

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "ps",
		Description: "list an app's processes",
		Usage:       "",
		Action:      cmdPs,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "app",
				Usage: "app name. Inferred from current directory if not specified.",
			},
		},
	})
}

func cmdPs(c *cli.Context) {
	app := c.String("app")

	if app == "" {
		app = DirAppName()
	}

	data, err := ConvoxGet(fmt.Sprintf("/apps/%s/processes", app))

	if err != nil {
		stdcli.Error(err)
		return
	}

	var processes *Processes
	err = json.Unmarshal(data, &processes)

	if err != nil {
		stdcli.Error(err)
		return
	}

	for _, ps := range *processes {
		fmt.Printf("%-10v %-3v\n", ps.Name, ps.Count)
	}
}
