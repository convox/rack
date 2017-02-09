package main

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/urfave/cli.v1"

	"github.com/convox/rack/cmd/convox/helpers"
	"github.com/convox/rack/cmd/convox/stdcli"
	"github.com/convox/rack/manifest"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "deploy",
		Description: "deploy an app to AWS",
		Usage:       "<directory>",
		Action:      cmdDeploy,
		Flags: append(
			buildCreateFlags,
			cli.BoolFlag{
				Name:   "wait",
				EnvVar: "CONVOX_WAIT",
				Usage:  "wait for release to finish promoting before returning",
			},
		),
	})
}

func cmdDeploy(c *cli.Context) error {
	wd := "."

	if len(c.Args()) > 0 {
		wd = c.Args()[0]
	}

	dir, app, err := stdcli.DirApp(c, wd)
	if err != nil {
		return stdcli.Error(err)
	}

	a, err := rackClient(c).GetApp(app)
	if err != nil {
		return stdcli.Error(err)
	}

	switch a.Status {
	case "creating":
		return stdcli.Error(fmt.Errorf("app %s is still being created", app))
	case "updating":
		return stdcli.Error(fmt.Errorf("app %s is being updated", app))
	}

	if !helpers.Exists(c.String("file")) {
		return stdcli.Error(fmt.Errorf("no docker-compose.yml found, try `convox init` to generate one"))
	}

	// validate docker-compose
	if _, err := manifest.LoadFile(c.String("file")); err != nil {
		return stdcli.Error(fmt.Errorf("invalid %s: %s", c.String("file"), strings.TrimSpace(err.Error())))
	}

	output := os.Stdout

	if c.Bool("id") {
		output = os.Stderr
	}

	output.Write([]byte(fmt.Sprintf("Deploying %s\n", app)))

	// build
	_, release, err := executeBuild(c, dir, app, c.String("file"), c.String("description"), output)
	if err != nil {
		return stdcli.Error(err)
	}

	if release == "" {
		return nil
	}

	output.Write([]byte(fmt.Sprintf("Release: %s\n", release)))

	if c.Bool("id") {
		os.Stdout.Write([]byte(release))
		output.Write([]byte("\n"))
	}

	output.Write([]byte(fmt.Sprintf("Promoting %s... ", release)))

	_, err = rackClient(c).PromoteRelease(app, release)
	if err != nil {
		return stdcli.Error(err)
	}

	output.Write([]byte("UPDATING\n"))

	if c.Bool("wait") {
		output.Write([]byte(fmt.Sprintf("Waiting for %s... ", release)))

		if err := waitForReleasePromotion(c, app, release); err != nil {
			return stdcli.Error(err)
		}

		output.Write([]byte("OK\n"))
	}

	return nil
}
