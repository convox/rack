package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/rack/cmd/convox/stdcli"
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

	a, err := rackClient(c).GetApp(app)

	if err != nil {
		stdcli.Error(err)
		return
	}

	releases, err := rackClient(c).GetReleases(app)

	if err != nil {
		stdcli.Error(err)
		return
	}

	t := stdcli.NewTable("ID", "CREATED", "STATUS")

	for _, r := range releases {
		status := ""

		if a.Release == r.Id {
			status = "active"
		}

		t.AddRow(r.Id, humanizeTime(r.Created), status)
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

	r, err := rackClient(c).GetRelease(app, release)

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

	fmt.Printf("Promoting %s... ", release)

	_, err = rackClient(c).PromoteRelease(app, release)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Println("UPDATING")
}
