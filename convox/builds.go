package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "builds",
		Description: "manage an app's builds",
		Usage:       "",
		Action:      cmdBuilds,
		Flags:       []cli.Flag{appFlag},
		Subcommands: []cli.Command{
			{
				Name:        "create",
				Description: "create a new build",
				Usage:       "",
				Action:      cmdBuildsCreate,
				Flags:       []cli.Flag{appFlag},
			},
		},
	})
}

func cmdBuilds(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	path := fmt.Sprintf("/apps/%s/builds", app)

	resp, err := ConvoxGet(path)

	if err != nil {
		stdcli.Error(err)
		return
	}

	var builds []Build

	err = json.Unmarshal(resp, &builds)

	if err != nil {
		stdcli.Error(err)
		return
	}

	longest := 7

	fmt.Printf(fmt.Sprintf("%%-12s  %%-%ds  %%-11s  %%-5s  %%s\n", longest), "ID", "RELEASE", "STATUS", "STARTED", "ENDED")

	var started Time
	var ended Time

	for _, build := range builds {
		started = build.Started
		ended = build.Ended
		fmt.Printf(fmt.Sprintf("%%-12s  %%-%ds  %%-11s  %%-5d  %%s\n", longest), build.Id, build.Release, started.Format(time.RFC822Z), ended.Format(time.RFC822Z), build.Ended)
	}
}

func cmdBuildsCreate(c *cli.Context) {
}
