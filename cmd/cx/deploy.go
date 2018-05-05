package main

import (
	"github.com/convox/rack/structs"
	"github.com/convox/stdcli"
)

func init() {
	CLI.Command("deploy", "create and promote a build", Deploy, stdcli.CommandOptions{
		Flags:    append(stdcli.OptionFlags(structs.BuildCreateOptions{}), flagApp, flagRack, flagWait),
		Usage:    "[dir]",
		Validate: stdcli.ArgsMax(1),
	})
}

func Deploy(c *stdcli.Context) error {
	b, err := build(c)
	if err != nil {
		return err
	}

	c.Startf("Promoting <release>%s</release>", b.Release)

	if err := releasePromote(c, app(c), b.Release); err != nil {
		return err
	}

	return c.OK()
}
