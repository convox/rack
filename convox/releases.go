package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"

	humanize "github.com/convox/cli/Godeps/_workspace/src/github.com/dustin/go-humanize"
)

type Release struct {
	Id string

	App string

	Build    string
	Env      string
	Manifest string

	Created time.Time
}

type Releases []Release

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "releases",
		Description: "list an app's releases",
		Usage:       "",
		Action:      cmdReleases,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "app",
				Usage: "App name. Inferred from current directory if not specified.",
			},
		},
		Subcommands: []cli.Command{
			{
				Name:        "promote",
				Description: "promote a release",
				Usage:       "<release id>",
				Action:      cmdReleasePromote,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "app",
						Usage: "app name. Inferred from current directory if not specified.",
					},
				},
			},
		},
	})
}

func cmdReleases(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	data, err := ConvoxGet("/apps/" + app)

	if err != nil {
		stdcli.Error(err)
		return
	}

	var a *App
	err = json.Unmarshal(data, &a)

	data, err = ConvoxGet(fmt.Sprintf("/apps/%s/releases", a.Name))

	if err != nil {
		stdcli.Error(err)
		return
	}

	var releases *Releases
	err = json.Unmarshal(data, &releases)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Printf("%-12s  %-12s  %-12s\n", "ID", "CREATED", "ACTIVE")

	for _, r := range *releases {
		active := ""
		if a.Parameters["Release"] == r.Id {
			active = "yes"
		}

		fmt.Printf("%-12s  %-12s  %-12s\n", r.Id, humanize.Time(r.Created), active)
	}
}

func cmdReleasePromote(c *cli.Context) {
	if len(c.Args()) < 1 {
		stdcli.Usage(c, "release promote")
		return
	}

	release := c.Args()[0]

	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	_, err = postRelease(app, release)

	if err != nil {
		stdcli.Error(err)
		return
	}
}

func postRelease(app, release string) (*App, error) {
	fmt.Print("Releasing... ")

	// promote release
	_, err := ConvoxPost(fmt.Sprintf("/apps/%s/releases/%s/promote", app, release), "")

	if err != nil {
		return nil, err
	}

	// poll for complete
	for {
		data, err := ConvoxGet(fmt.Sprintf("/apps/%s/status", app))

		if err != nil {
			return nil, err
		}

		if string(data) == "running" {
			break
		}

		time.Sleep(1 * time.Second)
	}

	data, err := ConvoxGet("/apps/" + app)

	if err != nil {
		return nil, err
	}

	var a *App
	err = json.Unmarshal(data, &a)

	if err != nil {
		return nil, err
	}

	fmt.Printf("OK, %s\n", a.Parameters["Release"])

	return a, nil
}
