package main

import (
	"encoding/json"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "info",
		Description: "see info about an app",
		Usage:       "convox info [--app name]",
		Action:      cmdInfo,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "app",
				Usage: "app name. Inferred from current directory if not specified.",
			},
		},
	})
}

func cmdInfo(c *cli.Context) {
	app := c.String("app")

	if app == "" {
		app = DirAppName()
	}

	data, err := ConvoxGet("/apps/" + app)

	if err != nil {
		stdcli.Error(err)
		return
	}

	var a *App
	err = json.Unmarshal(data, &a)

	if err != nil {
		stdcli.Error(err)
		return
	}

	a.PrintInfo()
}
