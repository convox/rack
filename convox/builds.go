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

	fmt.Printf("%-12s  %-12s  %-9s  %-22s  %s\n", "ID", "RELEASE", "STATUS", "STARTED", "ENDED")

	for _, build := range builds {
		started := build.Started
		ended := build.Ended
		fmt.Printf("%-12s  %-12s  %-9s  %-22s  %s\n", build.Id, build.Release, build.Status, started.Format(time.RFC822Z), ended.Format(time.RFC822Z))
	}
}

func cmdBuildsCreate(c *cli.Context) {
	wd := "."

	if len(c.Args()) > 0 {
		wd = c.Args()[0]
	}

	dir, app, err := stdcli.DirApp(c, wd)

	if err != nil {
		stdcli.Error(err)
		return
	}

	_, err = ConvoxGet(fmt.Sprintf("/apps/%s", app))

	if err != nil {
		stdcli.Error(err)
		return
	}

	release, err := executeBuild(dir, app)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Sprintln("Build complete. Release ID: %s", release)
}
