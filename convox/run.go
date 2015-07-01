package main

import (
	"fmt"
	"net/url"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "run",
		Description: "run a one-off process",
		Usage:       "convox run [--app myapp] ps cmd",
		Action:      cmdRun,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "app",
				Usage: "app name. Inferred from current directory if not specified.",
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

	if len(c.Args()) != 2 {
		stdcli.Usage(c, "run")
		return
	}

	ps := c.Args()[0]
	command := c.Args()[1]

	v := url.Values{}
	v.Set("command", command)
	_, err = ConvoxPostForm(fmt.Sprintf("/apps/%s/processes/%s/run", app, ps), v)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Printf("Running %s `%s`\n", ps, command)

	// fmt.Printf("%v %v %v\n", app, ps, command)
}
