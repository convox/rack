package main

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "scale",
		Description: "scale an app's processes",
		Usage:       "",
		Action:      cmdScale,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "app",
				Usage: "app name. Inferred from current directory if not specified.",
			},
			cli.IntFlag{
				Name:  "count",
				Value: 1,
				Usage: "number of processes to keep running for every process type.",
			},
			cli.IntFlag{
				Name:  "memory",
				Value: 256,
				Usage: "amount of memory, in megabytes, available to every process.",
			},
		},
	})
}

func cmdScale(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	v := url.Values{}

	if c.IsSet("count") {
		v.Set("count", c.String("count"))
	}

	if c.IsSet("memory") {
		v.Set("mem", c.String("memory"))
	}

	if len(v) > 0 {
		_, err = ConvoxPostForm("/apps/"+app, v)

		if err != nil {
			stdcli.Error(err)
			return
		}
	}

	data, err := ConvoxGet("/apps/" + app)

	if err != nil {
		stdcli.Error(err)
		return
	}

	var a *App
	err = json.Unmarshal(data, &a)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Printf("Count %v\n", a.Parameters["DesiredCount"])
	fmt.Printf("Memory %v\n", a.Parameters["Memory"])
}
