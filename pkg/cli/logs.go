package cli

import (
	"io"

	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/sdk"
	"github.com/convox/stdcli"
)

func init() {
	register("logs", "get logs for an app", Logs, stdcli.CommandOptions{
		Flags:    append(stdcli.OptionFlags(structs.LogsOptions{}), flagApp, flagNoFollow, flagRack),
		Validate: stdcli.Args(0),
	})
}

func Logs(rack sdk.Interface, c *stdcli.Context) error {
	var opts structs.LogsOptions

	if err := c.Options(&opts); err != nil {
		return err
	}

	if c.Bool("no-follow") {
		opts.Follow = options.Bool(false)
	}

	r, err := rack.AppLogs(app(c), opts)
	if err != nil {
		return err
	}

	_, err = io.Copy(c, r)

	return nil
}
