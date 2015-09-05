package main

import (
	"encoding/json"
	"fmt"
	"strings"
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
				Name:        "info",
				Description: "see info about a release",
				Usage:       "<release id>",
				Action:      cmdReleaseInfo,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "app",
						Usage: "app name. Inferred from current directory if not specified.",
					},
				},
			},
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

	t := stdcli.NewTable("ID", "CREATED", "ACTIVE")

	for _, r := range *releases {
		active := ""
		if a.Parameters["Release"] == r.Id {
			active = "yes"
		}

		t.AddRow(r.Id, humanize.Time(r.Created), active)
	}

	t.Print()
}

func cmdReleaseInfo(c *cli.Context) {
	if len(c.Args()) < 1 {
		stdcli.Usage(c, "release info")
		return
	}

	release := c.Args()[0]

	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	data, err := ConvoxGet(fmt.Sprintf("/apps/%s/releases/%s", app, release))

	if err != nil {
		stdcli.Error(err)
		return
	}

	var r Release

	err = json.Unmarshal(data, &r)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Printf("Id       %s\n", r.Id)
	fmt.Printf("Build    %s\n", r.Build)
	fmt.Printf("Created  %s\n", r.Created)
	fmt.Printf("Env      ")
	fmt.Println(strings.Replace(r.Env, "\n", "\n         ", -1))
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
	fmt.Printf("Promoting %s... ", release)

	// promote release
	_, err := ConvoxPost(fmt.Sprintf("/apps/%s/releases/%s/promote", app, release), "")

	if err != nil {
		return nil, err
	}

	// poll for complete
	for {
		data, err := ConvoxGet(fmt.Sprintf("/apps/%s", app))

		if err != nil {
			return nil, err
		}

		var app App

		err = json.Unmarshal(data, &app)

		if err != nil {
			return nil, err
		}

		if app.Status == "running" {
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
