package main

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "run",
		Description: "run a one-off command in your Convox rack",
		Usage:       "[process] [command]",
		Action:      cmdRun,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "app",
				Usage: "App name. Inferred from current directory if not specified.",
			},
		},
	})
}

func cmdRun(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	if len(c.Args()) < 1 {
		stdcli.Usage(c, "run")
		return
	}

	ps := c.Args()[0]

	command := ""

	if len(c.Args()) > 1 {
		args := c.Args()[1:]
		command = strings.Join(args, " ")
	}

	v := url.Values{}
	v.Set("command", command)

	_, err = ConvoxPostForm(fmt.Sprintf("/apps/%s/processes/%s/run", app, ps), v)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Printf("Running %s `%s`\n", ps, command)
}
