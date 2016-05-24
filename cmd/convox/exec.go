package main

import (
	"os"
	"strings"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/convox/rack/cmd/convox/stdcli"
	"gopkg.in/urfave/cli.v1"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "exec",
		Description: "exec a command in a process in your Convox rack",
		Usage:       "[pid] [command]",
		Action:      cmdExec,
		Flags:       []cli.Flag{appFlag},
	})
}

func cmdExec(c *cli.Context) error {
	fd := os.Stdin.Fd()
	stdinState, err := terminal.GetState(int(fd))
	if err != nil {
		return stdcli.ExitError(err)
	}
	defer terminal.Restore(int(fd), stdinState)

	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.ExitError(err)
	}

	if len(c.Args()) < 2 {
		stdcli.Usage(c, "exec")
		return nil
	}

	w, h, err := terminal.GetSize(int(fd))
	if err != nil {
		return stdcli.ExitError(err)
	}

	ps := c.Args()[0]

	code, err := rackClient(c).ExecProcessAttached(app, ps, strings.Join(c.Args()[1:], " "), os.Stdin, os.Stdout, h, w)
	terminal.Restore(int(fd), stdinState)
	if err != nil {
		return stdcli.ExitError(err)
	}

	return cli.NewExitError("", code)
}
