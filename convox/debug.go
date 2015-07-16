package main

import (
	"fmt"
	"strings"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "debug",
		Description: "get system events for debugging purposes",
		Usage:       "",
		Action:      cmdDebug,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "app",
				Usage: "app name. Inferred from current directory if not specified.",
			},
		},
	})
}

func cmdDebug(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	data, err := ConvoxGet(fmt.Sprintf("/apps/%s/events", app))

	if err != nil {
		stdcli.Error(err)
		return
	}

	lines := strings.Split(string(data), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		fmt.Printf("%v\n", lines[i])
	}
}
