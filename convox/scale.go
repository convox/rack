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
				Name:  "name",
				Usage: "app name. Inferred from current directory if not specified.",
			},
		},
	})
}

func cmdScale(c *cli.Context) {
	name := c.String("name")

	if name == "" {
		name = DirAppName()
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
		_, err = ConvoxPostForm("/apps/"+name, v)

		if err != nil {
			stdcli.Error(err)
			return
		}
	}

	data, err := ConvoxGet(fmt.Sprintf("/apps/%s", name))

	if err != nil {
		stdcli.Error(err)
		return
	}

	var app *App
	err = json.Unmarshal(data, &app)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Printf("Scale %v\n", app.Parameters["DesiredCount"])
}
