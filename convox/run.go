package main

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/docker/docker/pkg/term"

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
			cli.BoolFlag{
				Name:  "detach",
				Usage: "run in the background",
			},
		},
	})
}

func cmdRun(c *cli.Context) {
	if c.Bool("detach") {
		cmdRunDetached(c)
		return
	}

	fd := os.Stdin.Fd()

	oldState, err := term.SetRawTerminal(fd)

	if err != nil {
		stdcli.Error(err)
		return
	}

	defer term.RestoreTerminal(fd, oldState)

	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	if len(c.Args()) < 2 {
		stdcli.Usage(c, "run")
		return
	}

	ps := c.Args()[0]

	code, err := rackClient().RunProcessAttached(app, ps, strings.Join(c.Args()[1:], " "), os.Stdin, os.Stdout)

	if err != nil {
		stdcli.Error(err)
		return
	}

	os.Exit(code)
}

func cmdRunDetached(c *cli.Context) {
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

	fmt.Printf("Running `%s` on %s... ", command, ps)

	err = rackClient().RunProcessDetached(app, ps, command)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Println("OK")
}

var CodeRemoverRegex = regexp.MustCompile(`\x1b\[.n`)

type CodeStripper struct {
	writer io.Writer
}

func (cs CodeStripper) Write(data []byte) (int, error) {
	_, err := cs.writer.Write(CodeRemoverRegex.ReplaceAll(data, []byte("")))
	return len(data), err
}
