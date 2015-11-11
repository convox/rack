package main

import (
	"fmt"
	"os"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/rack/cmd/convox/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "logs",
		Description: "stream the logs for an application",
		Usage:       "",
		Action:      cmdLogsStream,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "app",
				Usage: "App name. Inferred from current directory if not specified.",
			},
		},
	})
}

func cmdLogsStream(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Println("sup dog 1")

	err = rackClient(c).StreamAppLogs(app, os.Stdout)

	fmt.Println("sup dog 2")

	if err != nil {
		stdcli.Error(err)
		return
	}
}
