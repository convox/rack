package main

import (
	"fmt"
	"strings"

	"gopkg.in/urfave/cli.v1"

	"github.com/convox/rack/cmd/convox/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "releases",
		Description: "list an app's releases",
		Usage:       "",
		Action:      cmdReleases,
		Flags:       []cli.Flag{appFlag, rackFlag},
		Subcommands: []cli.Command{
			{
				Name:        "info",
				Description: "see info about a release",
				Usage:       "<release id>",
				Action:      cmdReleaseInfo,
				Flags:       []cli.Flag{appFlag, rackFlag},
			},
			{
				Name:        "promote",
				Description: "promote a release",
				Usage:       "<release id>",
				Action:      cmdReleasePromote,
				Flags: []cli.Flag{
					appFlag,
					rackFlag,
					cli.BoolFlag{
						Name:   "wait",
						EnvVar: "CONVOX_WAIT",
						Usage:  "wait for release to finish promoting before returning",
					},
				},
			},
		},
	})
}

func cmdReleases(c *cli.Context) error {
	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.Error(err)
	}

	if len(c.Args()) > 0 {
		return stdcli.Error(fmt.Errorf("`convox releases` does not take arguments. Perhaps you meant `convox releases help`?"))
	}

	if c.Bool("help") {
		stdcli.Usage(c, "")
		return nil
	}

	a, err := rackClient(c).GetApp(app)
	if err != nil {
		return stdcli.Error(err)
	}

	releases, err := rackClient(c).GetReleases(app)
	if err != nil {
		return stdcli.Error(err)
	}

	t := stdcli.NewTable("ID", "CREATED", "BUILD", "STATUS")

	for _, r := range releases {
		status := ""

		if a.Release == r.Id {
			status = "active"
		}

		t.AddRow(r.Id, humanizeTime(r.Created), r.Build, status)
	}

	t.Print()
	return nil
}

func cmdReleaseInfo(c *cli.Context) error {
	if len(c.Args()) < 1 {
		stdcli.Usage(c, "release info")
		return nil
	}

	release := c.Args()[0]

	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.Error(err)
	}

	r, err := rackClient(c).GetRelease(app, release)
	if err != nil {
		return stdcli.Error(err)
	}

	fmt.Printf("Id       %s\n", r.Id)
	fmt.Printf("Build    %s\n", r.Build)
	fmt.Printf("Created  %s\n", r.Created)
	fmt.Printf("Env      ")

	fmt.Println(strings.Replace(r.Env, "\n", "\n         ", -1))
	return nil
}

func cmdReleasePromote(c *cli.Context) error {
	if len(c.Args()) < 1 {
		stdcli.Usage(c, "releases promote")
		return nil
	}

	release := c.Args()[0]

	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.Error(err)
	}

	fmt.Printf("Promoting %s... ", release)

	_, err = rackClient(c).PromoteRelease(app, release)
	if err != nil {
		return stdcli.Error(err)
	}

	fmt.Println("UPDATING")

	if c.Bool("wait") {
		fmt.Printf("Waiting for stabilization... ")

		if err := waitForReleasePromotion(c, app, release); err != nil {
			return stdcli.Error(err)
		}

		fmt.Println("OK")
	}

	return nil
}

func waitForReleasePromotion(c *cli.Context, app, release string) error {
	return waitForAppRunning(c, app)
}
