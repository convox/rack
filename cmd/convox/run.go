package main

import (
	"strings"

	"github.com/convox/rack/options"
	"github.com/convox/rack/structs"
	"github.com/convox/stdcli"
)

func init() {
	CLI.Command("run", "execute a command in a new process", Run, stdcli.CommandOptions{
		Flags:    append(stdcli.OptionFlags(structs.ProcessRunOptions{}), flagRack, flagApp),
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

	w, h, err := c.TerminalSize()
	if err != nil {
		return err
	}

	if err := c.TerminalRaw(); err != nil {
		return err
	}

	defer c.TerminalRestore()

	var opts structs.ProcessRunOptions

	if err := c.Options(&opts); err != nil {
		return err
	}

	// TODO version
	if s.Version <= "20180625222015" {
		opts.Command = options.String(strings.Join(c.Args[1:], " "))
		opts.Height = options.Int(h)
		opts.Width = options.Int(w)

		code, err := provider(c).ProcessRunAttached(app(c), service, c, opts)
		if err != nil {
			return err
		}

		return stdcli.Exit(code)
	}

	opts.Command = options.String("sleep 3600")

	ps, err := provider(c).ProcessRun(app(c), c.Arg(0), opts)
	if err != nil {
		return err
	}

	defer provider(c).ProcessStop(app(c), ps.Id)

	command := strings.Join(c.Args[1:], " ")

	eopts := structs.ProcessExecOptions{
		Height: options.Int(h),
		Width:  options.Int(w),
	}

	code, err := provider(c).ProcessExec(app(c), ps.Id, command, c, eopts)
	if err != nil {
		return err
	}

	return stdcli.Exit(code)
}
