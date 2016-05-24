package main

import (
	"fmt"

	"github.com/convox/rack/cmd/convox/stdcli"
	"gopkg.in/urfave/cli.v1"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "deploy",
		Description: "deploy an app to AWS",
		Usage:       "<directory>",
		Action:      cmdDeploy,
		Flags:       buildCreateFlags,
	})
}

func cmdDeploy(c *cli.Context) error {
	wd := "."

	if len(c.Args()) > 0 {
		wd = c.Args()[0]
	}

	dir, app, err := stdcli.DirApp(c, wd)
	if err != nil {
		return stdcli.ExitError(err)
	}

	fmt.Printf("Deploying %s\n", app)

	a, err := rackClient(c).GetApp(app)
	if err != nil {
		return stdcli.ExitError(err)
	}

	switch a.Status {
	case "creating":
		return stdcli.ExitError(fmt.Errorf("app is still creating: %s", app))
	case "running", "updating":
	default:
		return stdcli.ExitError(fmt.Errorf("unable to build app: %s", app))
	}

	// build
	release, err := executeBuild(c, dir, app, c.String("file"), c.String("description"))
	if err != nil {
		return stdcli.ExitError(err)
	}

	if release == "" {
		return nil
	}

	fmt.Printf("Promoting %s... ", release)

	_, err = rackClient(c).PromoteRelease(app, release)
	if err != nil {
		return stdcli.ExitError(err)
	}

	fmt.Println("UPDATING")
	return nil
}
