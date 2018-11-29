package cli

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/sdk"
	"github.com/convox/stdcli"
)

func init() {
	register("releases", "list releases for an app", Releases, stdcli.CommandOptions{
		Flags:    append(stdcli.OptionFlags(structs.ReleaseListOptions{}), flagRack, flagApp),
		Validate: stdcli.Args(0),
	})

	register("releases info", "get information about a release", ReleasesInfo, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagApp, flagRack},
		Validate: stdcli.Args(1),
	})

	register("releases manifest", "get manifest for a release", ReleasesManifest, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagApp, flagRack},
		Validate: stdcli.Args(1),
	})

	register("releases promote", "promote a release", ReleasesPromote, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagApp, flagRack, flagWait},
		Validate: stdcli.ArgsMax(1),
	})

	register("releases rollback", "copy an old release forward and promote it", ReleasesRollback, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagApp, flagId, flagRack, flagWait},
		Validate: stdcli.Args(1),
	})
}

func Releases(rack sdk.Interface, c *stdcli.Context) error {
	var opts structs.ReleaseListOptions

	if err := c.Options(&opts); err != nil {
		return err
	}

	a, err := rack.AppGet(app(c))
	if err != nil {
		return err
	}

	rs, err := rack.ReleaseList(app(c), opts)
	if err != nil {
		return err
	}

	t := c.Table("ID", "STATUS", "BUILD", "CREATED", "DESCRIPTION")

	for _, r := range rs {
		status := ""

		if a.Release == r.Id {
			status = "active"
		}

		t.AddRow(r.Id, status, r.Build, helpers.Ago(r.Created), r.Description)
	}

	return t.Print()
}

func ReleasesInfo(rack sdk.Interface, c *stdcli.Context) error {
	r, err := rack.ReleaseGet(app(c), c.Arg(0))
	if err != nil {
		return err
	}

	i := c.Info()

	i.Add("Id", r.Id)
	i.Add("Build", r.Build)
	i.Add("Created", r.Created.Format(time.RFC3339))
	i.Add("Description", r.Description)
	i.Add("Env", r.Env)

	return i.Print()
}

func ReleasesManifest(rack sdk.Interface, c *stdcli.Context) error {
	release := c.Arg(0)

	r, err := rack.ReleaseGet(app(c), release)
	if err != nil {
		return err
	}

	if r.Build == "" {
		return fmt.Errorf("no build for release: %s", release)
	}

	b, err := rack.BuildGet(app(c), r.Build)
	if err != nil {
		return err
	}

	fmt.Fprintf(c, "%s\n", strings.TrimSpace(b.Manifest))

	return nil
}

func ReleasesPromote(rack sdk.Interface, c *stdcli.Context) error {
	release := c.Arg(0)

	if release == "" {
		rs, err := rack.ReleaseList(app(c), structs.ReleaseListOptions{Limit: options.Int(1)})
		if err != nil {
			return err
		}

		if len(rs) == 0 {
			return fmt.Errorf("no releases to promote")
		}

		release = rs[0].Id
	}

	return releasePromote(rack, c, app(c), release)
}

func releasePromote(rack sdk.Interface, c *stdcli.Context, app, id string) error {
	if id == "" {
		return fmt.Errorf("no release to promote")
	}

	a, err := rack.AppGet(app)
	if err != nil {
		return err
	}

	if a.Status != "running" {
		c.Startf("Waiting for app to be ready")

		if err := waitForAppRunning(rack, app); err != nil {
			return err
		}

		c.OK()
	}

	m, _, err := helpers.ReleaseManifest(rack, app, id)
	if err != nil {
		return err
	}

	c.Writef("Running hooks: <system>before-promote</system>\n")

	for _, s := range m.Services {
		if s.Hooks.BeforePromote != "" {
			opts := structs.ProcessRunOptions{
				Command: options.String(s.Hooks.BeforePromote),
				Release: options.String(id),
			}

			c.Writef("<service>%s</service>: <command>%s</command>\n", s.Name, s.Hooks.BeforePromote)

			code, err := runAttached(c, rack, app, s.Name, opts, 3600)
			if err != nil {
				return err
			}

			if code > 0 {
				return fmt.Errorf("exit %d", code)
			}

			c.OK()
		}
	}

	c.Startf("Promoting <release>%s</release>", id)

	if err := rack.ReleasePromote(app, id, structs.ReleasePromoteOptions{}); err != nil {
		return err
	}

	if err := waitForAppWithLogs(rack, c, app); err != nil {
		return err
	}

	c.OK()

	c.Writef("Running hooks: <system>after-promote</system>\n")

	for _, s := range m.Services {
		if s.Hooks.AfterPromote != "" {
			opts := structs.ProcessRunOptions{
				Command: options.String(s.Hooks.AfterPromote),
				Release: options.String(id),
			}

			c.Writef("<service>%s</service>: <command>%s</command>\n", s.Name, s.Hooks.AfterPromote)

			code, err := runAttached(c, rack, app, s.Name, opts, 3600)
			if err != nil {
				return err
			}

			if code > 0 {
				c.Writef("<error>exit %d</error>\n", code)

				c.Startf("Rolling back to <release>%s</release>", a.Release)

				if err := rack.ReleasePromote(app, id, structs.ReleasePromoteOptions{}); err != nil {
					return err
				}

				if err := waitForAppWithLogs(rack, c, app); err != nil {
					return err
				}

				c.OK()

				return fmt.Errorf("hook failure on service %s", s.Name)
			}

			c.OK()
		}
	}

	return nil
}

func ReleasesRollback(rack sdk.Interface, c *stdcli.Context) error {
	var stdout io.Writer

	if c.Bool("id") {
		stdout = c.Writer().Stdout
		c.Writer().Stdout = c.Writer().Stderr
	}

	release := c.Arg(0)

	c.Startf("Rolling back to <release>%s</release>", release)

	ro, err := rack.ReleaseGet(app(c), release)
	if err != nil {
		return err
	}

	rn, err := rack.ReleaseCreate(app(c), structs.ReleaseCreateOptions{
		Build: options.String(ro.Build),
		Env:   options.String(ro.Env),
	})
	if err != nil {
		return err
	}

	c.OK(rn.Id)

	if err := releasePromote(rack, c, app(c), rn.Id); err != nil {
		return err
	}

	if c.Bool("id") {
		fmt.Fprintf(stdout, rn.Id)
	}

	return nil
}
