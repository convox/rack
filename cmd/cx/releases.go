package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/convox/rack/helpers"
	"github.com/convox/rack/options"
	"github.com/convox/rack/structs"
	"github.com/convox/stdcli"
)

func init() {
	CLI.Command("releases", "list releases", Releases, stdcli.CommandOptions{
		Flags:    append(stdcli.OptionFlags(structs.ReleaseListOptions{}), flagRack, flagApp),
		Validate: stdcli.Args(0),
	})

	CLI.Command("releases info", "get information about a release", ReleasesInfo, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagApp, flagRack},
		Validate: stdcli.Args(1),
	})

	CLI.Command("releases manifest", "get manifest for a release", ReleasesManifest, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagApp, flagRack},
		Validate: stdcli.Args(1),
	})

	CLI.Command("releases promote", "promote a release", ReleasesPromote, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagApp, flagRack, flagWait},
		Validate: stdcli.Args(1),
	})

	CLI.Command("releases rollback", "copy an old release forward and promote it", ReleasesRollback, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagApp, flagRack},
		Validate: stdcli.Args(1),
	})
}

func Releases(c *stdcli.Context) error {
	var opts structs.ReleaseListOptions

	if err := c.Options(&opts); err != nil {
		return err
	}

	a, err := provider(c).AppGet(app(c))
	if err != nil {
		return err
	}

	rs, err := provider(c).ReleaseList(app(c), opts)
	if err != nil {
		return err
	}

	t := c.Table("ID", "STATUS", "BUILD", "CREATED")

	for _, r := range rs {
		status := ""

		if a.Release == r.Id {
			status = "active"
		}

		t.AddRow(r.Id, status, r.Build, helpers.Ago(r.Created))
	}

	return t.Print()
}

func ReleasesInfo(c *stdcli.Context) error {
	r, err := provider(c).ReleaseGet(app(c), c.Arg(0))
	if err != nil {
		return err
	}

	i := c.Info()

	i.Add("Id", r.Id)
	i.Add("Build", r.Build)
	i.Add("Created", r.Created.Format(time.RFC3339))
	i.Add("Env", r.Env)

	return i.Print()
}

func ReleasesManifest(c *stdcli.Context) error {
	release := c.Arg(0)

	r, err := provider(c).ReleaseGet(app(c), release)
	if err != nil {
		return err
	}

	if r.Build == "" {
		return fmt.Errorf("no build for release: %s", release)
	}

	b, err := provider(c).BuildGet(app(c), r.Build)
	if err != nil {
		return err
	}

	fmt.Fprintf(c, "%s\n", strings.TrimSpace(b.Manifest))

	return nil
}

func ReleasesPromote(c *stdcli.Context) error {
	return releasePromote(c, app(c), c.Arg(0))
}

func releasePromote(c *stdcli.Context, app, id string) error {
	c.Startf("Promoting <release>%s</release>", id)

	if err := provider(c).ReleasePromote(app, id); err != nil {
		return err
	}

	if c.Bool("wait") {
		if err := waitForAppWithLogs(c, app); err != nil {
			return err
		}
	}

	return c.OK()
}

func ReleasesRollback(c *stdcli.Context) error {
	release := c.Arg(0)

	c.Startf("Rolling back to <release>%s</release>", release)

	ro, err := provider(c).ReleaseGet(app(c), release)
	if err != nil {
		return err
	}

	rn, err := provider(c).ReleaseCreate(app(c), structs.ReleaseCreateOptions{
		Build: options.String(ro.Build),
		Env:   options.String(ro.Env),
	})
	if err != nil {
		return err
	}

	c.OK(rn.Id)

	c.Startf("Promoting <release>%s</release>", rn.Id)

	if err := provider(c).ReleasePromote(app(c), rn.Id); err != nil {
		return err
	}

	return c.OK()
}
