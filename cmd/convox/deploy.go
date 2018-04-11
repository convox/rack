package main

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/urfave/cli.v1"

	"github.com/convox/rack/cmd/convox/stdcli"
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
	stdcli.NeedHelp(c)
	wd := "."

	if len(c.Args()) > 0 {
		stdcli.NeedArg(c, 1)
		wd = c.Args()[0]
	}

	dir, app, err := stdcli.DirApp(c, wd)
	if err != nil {
		return stdcli.Error(err)
	}

	a, err := rackClient(c).GetApp(app)
	if err != nil {
		if strings.Contains(err.Error(), "no such app") {
			return stdcli.Errorf("%s, try running `convox apps create`", err.Error())
		}
		return stdcli.Error(err)
	}

	if a.Status != "running" {
		return stdcli.Error(fmt.Errorf("unable to deploy %s in a non-running status: %s", app, a.Status))
	}

	output := os.Stdout

	if c.Bool("id") {
		output = os.Stderr
	}

	output.Write([]byte(fmt.Sprintf("Deploying %s\n", app)))

	// build
	build, release, err := executeBuild(c, dir, app, c.String("file"), c.String("description"), output)
	if err != nil {
		return stdcli.Error(err)
	}

	if release == "" {
		return stdcli.Error(fmt.Errorf("build %s is completed but missing release information", build))
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
		if err := waitForReleasePromotion(output, c, app, release); err != nil {
			return stdcli.Error(err)
		}
	}

	return nil
}
