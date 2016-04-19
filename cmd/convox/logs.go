package main

import (
	"os"
	"time"

	"github.com/codegangsta/cli"
	"github.com/convox/rack/cmd/convox/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "logs",
		Description: "stream the logs for an application",
		Usage:       "",
		Action:      cmdLogsStream,
		Flags: []cli.Flag{
			appFlag,
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

func cmdLogsStream(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	err = rackClient(c).StreamAppLogs(app, c.String("filter"), c.BoolT("follow"), c.Duration("since"), os.Stdout)

	if err != nil {
		stdcli.Error(err)
		return
	}
}
