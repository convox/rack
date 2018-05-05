package main

import (
	"io"

	"github.com/convox/rack/structs"
	"github.com/convox/stdcli"
)

func init() {
	CLI.Command("logs", "get logs for an app", Logs, stdcli.CommandOptions{
		Flags:    append(stdcli.OptionFlags(structs.LogsOptions{}), flagRack, flagApp),
		Validate: stdcli.Args(0),
	})
}

func Logs(c *stdcli.Context) error {
	var opts structs.LogsOptions

	if err := c.Options(&opts); err != nil {
		return err
	}

	r, err := provider(c).AppLogs(app(c), opts)
	if err != nil {
		return err
	}

	io.Copy(c, r)

	return nil
}
