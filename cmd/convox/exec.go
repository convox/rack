package main

import (
	"strings"

	"github.com/convox/rack/options"
	"github.com/convox/rack/structs"
	"github.com/convox/stdcli"
)

func init() {
	CLI.Command("exec", "execute a command in a running process", Exec, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack, flagApp},
		Usage:    "<pid> <command>",
		Validate: stdcli.ArgsMin(2),
	})
}

func Exec(c *stdcli.Context) error {
	pid := c.Arg(0)
	command := strings.Join(c.Args[1:], " ")

	w, h, err := c.TerminalSize()
	if err != nil {
		return err
	}

	opts := structs.ProcessExecOptions{
		Height: options.Int(h),
		Width:  options.Int(w),
	}

	if err := c.TerminalRaw(); err != nil {
		return err
	}

	defer c.TerminalRestore()

	code, err := provider(c).ProcessExec(app(c), pid, command, c, opts)
	if err != nil {
		return err
	}

	return stdcli.Exit(code)
}
