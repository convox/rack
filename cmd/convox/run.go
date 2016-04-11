package main

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/codegangsta/cli"
	"github.com/convox/rack/cmd/convox/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "run",
		Description: "run a one-off command in your Convox rack",
		Usage:       "[process] [command]",
		Action:      cmdRun,
		Flags: []cli.Flag{appFlag,
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

func cmdRun(c *cli.Context) {
	if c.Bool("detach") {
		cmdRunDetached(c)
		return
	}

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

	args := strings.Join(c.Args()[1:], " ")

	release := c.String("release")

	code, err := runAttached(c, app, ps, args, release)

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

	release := c.String("release")

	fmt.Printf("Running `%s` on %s... ", command, ps)

	err = rackClient(c).RunProcessDetached(app, ps, command, release)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Println("OK")
}

func runAttached(c *cli.Context, app, ps, args, release string) (int, error) {
	fd := os.Stdin.Fd()

	if terminal.IsTerminal(int(fd)) {
		stdinState, err := terminal.GetState(int(fd))

		if err != nil {
			return -1, err
		}

		defer terminal.Restore(int(fd), stdinState)
	}

	w, h, err := terminal.GetSize(int(fd))

	if err != nil {
		return -1, err
	}

	code, err := rackClient(c).RunProcessAttached(app, ps, args, release, h, w, os.Stdin, os.Stdout)

	if err != nil {
		return -1, err
	}

	return code, nil
}

var CodeRemoverRegex = regexp.MustCompile(`\x1b\[.n`)

type CodeStripper struct {
	writer io.Writer
}

func (cs CodeStripper) Write(data []byte) (int, error) {
	_, err := cs.writer.Write(CodeRemoverRegex.ReplaceAll(data, []byte("")))
	return len(data), err
}
