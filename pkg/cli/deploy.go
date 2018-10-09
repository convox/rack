package cli

import (
	"fmt"
	"io"

	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/sdk"
	"github.com/convox/stdcli"
)

func init() {
	register("deploy", "create and promote a build", Deploy, stdcli.CommandOptions{
		Flags:    append(stdcli.OptionFlags(structs.BuildCreateOptions{}), flagApp, flagId, flagRack, flagWait),
		Usage:    "[dir]",
		Validate: stdcli.ArgsMax(1),
	})
}

func Deploy(rack sdk.Interface, c *stdcli.Context) error {
	var stdout io.Writer

	if c.Bool("id") {
		stdout = c.Writer().Stdout
		c.Writer().Stdout = c.Writer().Stderr
	}

	b, err := build(rack, c, false)
	if err != nil {
		return err
	}

	if err := releasePromote(rack, c, app(c), b.Release); err != nil {
		return err
	}

	if c.Bool("id") {
		fmt.Fprintf(stdout, b.Release)
	}

	return nil
}
