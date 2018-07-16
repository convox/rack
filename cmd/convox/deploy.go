package main

import (
	"fmt"
	"io"

	"github.com/convox/rack/structs"
	"github.com/convox/stdcli"
)

func init() {
	CLI.Command("deploy", "create and promote a build", Deploy, stdcli.CommandOptions{
		Flags:    append(stdcli.OptionFlags(structs.BuildCreateOptions{}), flagApp, flagId, flagRack, flagWait),
		Usage:    "[dir]",
		Validate: stdcli.ArgsMax(1),
	})
}

func Deploy(c *stdcli.Context) error {
	var stdout io.Writer

	if c.Bool("id") {
		stdout = c.Writer().Stdout
		c.Writer().Stdout = c.Writer().Stderr
	}

	b, err := build(c)
	if err != nil {
		return err
	}

	if err := releasePromote(c, app(c), b.Release); err != nil {
		return err
	}

	if c.Bool("id") {
		fmt.Fprintf(stdout, b.Release)
	}

	return nil
}
