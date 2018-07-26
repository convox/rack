package main

import (
	"strings"

	"github.com/convox/rack/options"
	"github.com/convox/rack/structs"
	"github.com/convox/stdcli"
)

func init() {
	CLI.Command("run", "execute a command in a new process", Run, stdcli.CommandOptions{
		Flags: append(
			stdcli.OptionFlags(structs.ProcessRunOptions{}),
			flagRack,
			flagApp,
			stdcli.BoolFlag("detach", "d", "run process in the background"),
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

	var width, height int

	if c.Reader().IsTerminal() {
		if err := c.TerminalRaw(); err != nil {
			return err
		}

		defer c.TerminalRestore()
	}

	if w, h, err := c.TerminalSize(); err == nil {
		width = w
		height = h
	}

	var opts structs.ProcessRunOptions

	if err := c.Options(&opts); err != nil {
		return err
	}

	if s.Version <= "20180708231844" {
		if c.Bool("detach") {
			c.Startf("Running detached process")

			opts.Command = options.String(strings.Join(c.Args[1:], " "))

			pid, err := provider(c).ProcessRunDetached(app(c), service, opts)
			if err != nil {
				return err
			}

			return c.OK(pid)
		}

		opts.Command = options.String(strings.Join(c.Args[1:], " "))

		if height > 0 && width > 0 {
			opts.Height = options.Int(height)
			opts.Width = options.Int(width)
		}

		code, err := provider(c).ProcessRunAttached(app(c), service, c, opts)
		if err != nil {
			return err
		}

		return stdcli.Exit(code)
	}

	if c.Bool("detach") {
		c.Startf("Running detached process")

		opts.Command = options.String(strings.Join(c.Args[1:], " "))

		ps, err := provider(c).ProcessRun(app(c), service, opts)
		if err != nil {
			return err
		}

		return c.OK(ps.Id)
	}

	opts.Command = options.String("sleep 3600")

	ps, err := provider(c).ProcessRun(app(c), c.Arg(0), opts)
	if err != nil {
		return err
	}

	defer provider(c).ProcessStop(app(c), ps.Id)

	if err := waitForProcessRunning(c, app(c), ps.Id); err != nil {
		return err
	}

	command := strings.Join(c.Args[1:], " ")

	eopts := structs.ProcessExecOptions{
		Entrypoint: options.Bool(true),
	}

	if height > 0 && width > 0 {
		eopts.Height = options.Int(height)
		eopts.Width = options.Int(width)
	}

	code, err := provider(c).ProcessExec(app(c), ps.Id, command, c, eopts)
	if err != nil {
		return err
	}

	return stdcli.Exit(code)
}
