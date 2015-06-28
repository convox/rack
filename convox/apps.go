package main

import (
	"encoding/json"
	"fmt"

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
		Description: "list apps",
		Action:      cmdApps,
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
