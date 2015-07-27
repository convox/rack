package main

import (
	"encoding/json"
	"fmt"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "system",
		Description: "manage the base convox system",
		Usage:       "",
		Action:      cmdSystem,
		Subcommands: []cli.Command{
			{
				Name:        "update",
				Description: "update the convox system API",
				Usage:       "[version]",
				Action:      cmdSystemUpate,
			},
			{
				Name:        "scale",
				Description: "scale the convox system cluster",
				Usage:       "",
				Action:      cmdSystemScale,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "type",
						Usage: "instance type, e.g. t2.small or c3.xlargs",
					},
					cli.IntFlag{
						Name:  "num",
						Usage: "instance number, e.g. 3 or 10",
					},
				},
			},
		},
	})
}

func cmdSystem(c *cli.Context) {
	data, err := ConvoxGet("/system")

	if err != nil {
		stdcli.Error(err)
		return
	}

	var a *App
	err = json.Unmarshal(data, &a)

	fmt.Printf("%+v\n", a)
}

func cmdSystemUpate(c *cli.Context) {
	stdcli.Error(fmt.Errorf("not implemented"))
}

func cmdSystemScale(c *cli.Context) {
	stdcli.Error(fmt.Errorf("not implemented"))
}
