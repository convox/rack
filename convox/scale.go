package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "scale",
		Description: "scale an app's processes",
		Usage:       "<count>",
		Action:      cmdScale,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "app",
				Usage: "app name. Inferred from current directory if not specified.",
			},
		},
	})
}

func cmdScale(c *cli.Context) {
	app := c.String("app")

	if app == "" {
		app = DirAppName()
	}

	if len(c.Args()) == 1 {
		count := c.Args()[0]
		_, err := strconv.Atoi(count)

		if err != nil {
			stdcli.Error(fmt.Errorf("Count must be numeric."))
			return
		}

		v := url.Values{}
		v.Set("count", count)
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

	fmt.Printf("Scale %v\n", a.Parameters["DesiredCount"])
}
