package main

import (
	"fmt"
	"strings"

	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/stdcli"
)

func init() {
	CLI.Command("run", "execute a command in a new process", Run, stdcli.CommandOptions{
		Flags: append(
			stdcli.OptionFlags(structs.ProcessRunOptions{}),
			flagRack,
			flagApp,
			stdcli.BoolFlag("detach", "d", "run process in the background"),
			stdcli.IntFlag("timeout", "t", "timeout"),
		),
		Usage:    "<service> <command>",
		Validate: stdcli.ArgsMin(2),
	})
}

func Run(c *stdcli.Context) error {
	s, err := provider(c).SystemGet()
	if err != nil {
		return err
	}

	service := c.Arg(0)
	command := strings.Join(c.Args[1:], " ")

	var opts structs.ProcessRunOptions

	if err := c.Options(&opts); err != nil {
		return err
	}

	timeout := 3600

	if t := c.Int("timeout"); t > 0 {
		timeout = t
	}

	restore := c.TerminalRaw()
	defer restore()

	if s.Version <= "20180708231844" {
		opts.Command = options.String(command)

		if c.Bool("detach") {
			c.Startf("Running detached process")

			pid, err := provider(c).ProcessRunDetached(app(c), service, opts)
			if err != nil {
				return err
			}

			return c.OK(pid)
		}

		code, err := provider(c).ProcessRunAttached(app(c), service, c, timeout, opts)
		if err != nil {
			return err
		}

		return stdcli.Exit(code)
	}

	if c.Bool("detach") {
		c.Startf("Running detached process")

		ps, err := provider(c).ProcessRun(app(c), service, opts)
		if err != nil {
			return err
		}

		return c.OK(ps.Id)
	}

	opts.Command = options.String(fmt.Sprintf("sleep %d", timeout))

	ps, err := provider(c).ProcessRun(app(c), c.Arg(0), opts)
	if err != nil {
		return err
	}

	defer provider(c).ProcessStop(app(c), ps.Id)

	if err := waitForProcessRunning(c, app(c), ps.Id); err != nil {
		return err
	}

	eopts := structs.ProcessExecOptions{
		Entrypoint: options.Bool(true),
		Height:     opts.Height,
		Width:      opts.Width,
	}

	code, err := provider(c).ProcessExec(app(c), ps.Id, command, c, eopts)
	if err != nil {
		return err
	}

	return stdcli.Exit(code)
}
