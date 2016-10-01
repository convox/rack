package main

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/convox/rack/cmd/convox/stdcli"
	"gopkg.in/urfave/cli.v1"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "run",
		Description: "run a one-off command in your Convox rack",
		Usage:       "[process] [command]",
		Action:      cmdRun,
		Flags: []cli.Flag{
			appFlag,
			rackFlag,
			cli.BoolFlag{
				Name:  "detach",
				Usage: "run in the background",
			},
			cli.StringFlag{
				Name:  "release, r",
				Usage: "Release Name. Defaults to current release.",
			},
		},
	})
}

func cmdRun(c *cli.Context) error {
	if c.Bool("detach") {
		return cmdRunDetached(c)
	}

	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.Error(err)
	}

	if len(c.Args()) < 2 {
		stdcli.Usage(c, "run")
		return nil
	}

	ps := c.Args()[0]
	err = validateProcessId(c, app, ps)
	if err != nil {
		return stdcli.Error(err)
	}

	args := strings.Join(c.Args()[1:], " ")

	release := c.String("release")

	code, err := runAttached(c, app, ps, args, release)
	if err != nil {
		return stdcli.Error(err)
	}

	return cli.NewExitError("", code)
}

func cmdRunDetached(c *cli.Context) error {
	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return err
	}

	if len(c.Args()) < 1 {
		stdcli.Usage(c, "run")
		return nil
	}

	ps := c.Args()[0]
	err = validateProcessId(c, app, ps)
	if err != nil {
		return stdcli.Error(err)
	}

	command := ""

	if len(c.Args()) > 1 {
		args := c.Args()[1:]
		command = strings.Join(args, " ")
	}

	release := c.String("release")

	fmt.Printf("Running `%s` on %s... ", command, ps)

	err = rackClient(c).RunProcessDetached(app, ps, command, release)
	if err != nil {
		return stdcli.Error(err)
	}

	fmt.Println("OK")
	return nil
}

func runAttached(c *cli.Context, app, ps, args, release string) (int, error) {
	fd := os.Stdin.Fd()

	var w, h int

	if terminal.IsTerminal(int(fd)) {
		stdinState, err := terminal.GetState(int(fd))
		if err != nil {
			return -1, err
		}

		defer terminal.Restore(int(fd), stdinState)

		w, h, err = terminal.GetSize(int(fd))
		if err != nil {
			return -1, err
		}
	}

	code, err := rackClient(c).RunProcessAttached(app, ps, args, release, h, w, os.Stdin, os.Stdout)
	if err != nil {
		return -1, err
	}

	return code, nil
}

func validateProcessId(c *cli.Context, app, ps string) error {
	formation, err := rackClient(c).ListFormation(app)
	if err != nil {
		return err
	}

	for _, f := range formation {
		if ps == f.Name {
			return nil
		}
	}

	return fmt.Errorf("Unknown process name: %s", ps)
}
