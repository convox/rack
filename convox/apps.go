package main

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
)

type App struct {
	Name string

	Status     string
	Repository string

	Outputs    map[string]string
	Parameters map[string]string
	Tags       map[string]string
}

type Apps []App

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "apps",
		Action:      cmdApps,
		Description: "list apps",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "app",
				Usage: "app name. If not specified, use current directory.",
			},
		},
		Subcommands: []cli.Command{
			{
				Name:   "create",
				Usage:  "convox apps create [name]",
				Action: cmdAppCreate,
			},
		},
	})
}

func cmdApps(c *cli.Context) {
	data, err := ConvoxGet("/apps")

	if err != nil {
		stdcli.Error(err)
		return
	}

	var apps *Apps
	err = json.Unmarshal(data, &apps)

	if err != nil {
		stdcli.Error(err)
		return
	}

	for _, app := range *apps {
		fmt.Printf("%s\n", app.Name)
	}
}

func cmdAppCreate(c *cli.Context) {
	name := c.Args()[0]

	if name == "" {
		name = DirAppName()
	}

	v := url.Values{}
	v.Set("name", name)
	data, err := ConvoxPostForm("/apps", v)

	if err != nil {
		stdcli.Error(err)
		return
	}

	data, err = ConvoxGet("/apps/" + name)

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

	fmt.Printf("Created %s.\n", app.Name)
}
