package main

import (
	"fmt"
	"strings"
	"time"

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
						Name:  "wait",
						Usage: "wait for release to finish promoting before returning",
					},
				},
			},
		},
	})
}

func cmdReleases(c *cli.Context) error {
	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.ExitError(err)
	}

	if len(c.Args()) > 0 {
		return stdcli.ExitError(fmt.Errorf("`convox releases` does not take arguments. Perhaps you meant `convox registries info`?"))
	}

	if c.Bool("help") {
		stdcli.Usage(c, "")
		return nil
	}

	a, err := rackClient(c).GetApp(app)
	if err != nil {
		return stdcli.ExitError(err)
	}

	releases, err := rackClient(c).GetReleases(app)
	if err != nil {
		return stdcli.ExitError(err)
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
		return stdcli.ExitError(err)
	}

	r, err := rackClient(c).GetRelease(app, release)
	if err != nil {
		return stdcli.ExitError(err)
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
		return stdcli.ExitError(err)
	}

	fmt.Printf("Promoting %s... ", release)

	_, err = rackClient(c).PromoteRelease(app, release)
	if err != nil {
		return stdcli.ExitError(err)
	}

	fmt.Println("UPDATING")

	if c.Bool("wait") {
		fmt.Printf("Waiting for %s... ", release)

		if err := waitForReleasePromotion(c, app, release); err != nil {
			stdcli.ExitError(err)
		}

		fmt.Println("OK")
	}

	return nil
}

func waitForReleasePromotion(c *cli.Context, app, release string) error {
	if err := waitForAppRunning(c, app); err != nil {
		return err
	}

	form, err := rackClient(c).ListFormation(app)
	if err != nil {
		return err
	}

	desired := map[string]int{}

	for _, f := range form {
		desired[f.Name] = f.Count
	}

	tick := time.Tick(5 * time.Second)
	timeout := time.Tick(10 * time.Minute)

	for {
		select {
		case <-tick:
			pss, err := rackClient(c).GetProcesses(app, false)
			if err != nil {
				return err
			}

			found := map[string]int{}

			ready := true

			for _, p := range pss {
				if p.Release == release && p.Id != "" && p.Id != "pending" && !p.Started.IsZero() {
					found[p.Name] += 1
				} else {
					ready = false
					break
				}
			}

			for name, count := range found {
				if desired[name] != count {
					ready = false
					break
				}
			}

			if ready {
				return nil
			}
		case <-timeout:
			return fmt.Errorf("timeout")
		}
	}

	return nil
}
