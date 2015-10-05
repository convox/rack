package main

import (
	"os"
	"strings"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/codegangsta/cli"
	"github.com/convox/rack/cmd/convox/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "exec",
		Description: "exec a command in a process in your Convox rack",
		Usage:       "[pid] [command]",
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
	fd := os.Stdin.Fd()
	stdinState, err := terminal.GetState(int(fd))
	defer terminal.Restore(int(fd), stdinState)

	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	if len(c.Args()) < 2 {
		stdcli.Usage(c, "exec")
		return
	}

	ps := c.Args()[0]

	code, err := rackClient(c).ExecProcessAttached(app, ps, strings.Join(c.Args()[1:], " "), os.Stdin, os.Stdout)
	terminal.Restore(int(fd), stdinState)

	if err != nil {
		stdcli.Error(err)
		return
	}

	os.Exit(code)
}
