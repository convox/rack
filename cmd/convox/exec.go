package main

import (
	"fmt"
	"strings"

	"/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "exec",
		Description: "run a one-off command in a local Convox process",
		Usage:       "[process] [command]",
		Action:      cmdExec,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "app",
				Usage: "App name. Inferred from current directory if not specified.",
			},
		},
	})
}

func cmdExec(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	if len(c.Args()) < 1 {
		stdcli.Usage(c, "exec")
		return
	}

	ps := c.Args()[0]

	command := ""

	if len(c.Args()) > 1 {
		args := c.Args()[1:]
		command = strings.Join(args, " ")
	}

	err = stdcli.Run("docker", "exec", "-it", fmt.Sprintf("%s-%s", app, ps), "sh", "-c", command)

	if err != nil {
		stdcli.Error(err)
		return
	}
}
