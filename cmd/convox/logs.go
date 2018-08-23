package main

import (
	"io"

	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/stdcli"
)

func init() {
	CLI.Command("logs", "get logs for an app", Logs, stdcli.CommandOptions{
		Flags:    append(stdcli.OptionFlags(structs.LogsOptions{}), flagApp, flagNoFollow, flagRack),
		Validate: stdcli.Args(0),
	})
}

func Logs(c *stdcli.Context) error {
	var opts structs.LogsOptions

	if err := c.Options(&opts); err != nil {
		return err
	}

	if c.Bool("no-follow") {
		opts.Follow = options.Bool(false)
	}

	r, err := provider(c).AppLogs(app(c), opts)
	if err != nil {
		return err
	}

	_, err = io.Copy(c, r)

	return nil
}
