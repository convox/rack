package main

import (
	"fmt"
	"os"
	"time"

	"github.com/convox/rack/cmd/convox/stdcli"
	"gopkg.in/urfave/cli.v1"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "logs",
		Description: "stream the logs for an application",
		Usage:       "",
		Action:      cmdLogsStream,
		Flags: []cli.Flag{
			appFlag,
			rackFlag,
			cli.StringFlag{
				Name:  "filter",
				Usage: "Only return logs that match a filter pattern. If not specified, return all logs.",
			},
			cli.BoolTFlag{
				Name:  "follow",
				Usage: "Follow log output (default).",
			},
			cli.DurationFlag{
				Name:  "since",
				Usage: "Show logs since a duration, e.g. 10m or 1h2m10s.",
				Value: 2 * time.Minute,
			},
		},
	})
}

func cmdLogsStream(c *cli.Context) error {
	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.ExitError(err)
	}

	if len(c.Args()) > 0 {
		return stdcli.ExitError(fmt.Errorf("`convox logs` does not take arguments. Perhaps you meant `convox logs`?"))
	}

	err = rackClient(c).StreamAppLogs(app, c.String("filter"), c.BoolT("follow"), c.Duration("since"), os.Stdout)
	if err != nil {
		return stdcli.ExitError(err)
	}
	return nil
}
