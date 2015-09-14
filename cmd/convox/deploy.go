package main

import (
	"fmt"

	"github.com/convox/rack/cmd/convox/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/rack/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "deploy",
		Description: "deploy an app to AWS",
		Usage:       "<directory>",
		Action:      cmdDeploy,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "app",
				Usage: "App name. Inferred from current directory if not specified.",
			},
		},
	})
}

func cmdDeploy(c *cli.Context) {
	wd := "."

	if len(c.Args()) > 0 {
		wd = c.Args()[0]
	}

	dir, app, err := stdcli.DirApp(c, wd)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Printf("Deploying %s\n", app)

	_, err = rackClient(c).GetApp(app)

	if err != nil {
		stdcli.Error(err)
		return
	}

	// build
	release, err := executeBuild(c, dir, app)

	if err != nil {
		stdcli.Error(err)
		return
	}

	if release == "" {
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
