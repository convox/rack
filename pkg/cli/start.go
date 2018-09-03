package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/convox/rack/pkg/start"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/sdk"
	"github.com/convox/stdcli"
)

func init() {
	register("start", "start an application for local development", Start, stdcli.CommandOptions{
		Flags: []stdcli.Flag{
			flagRack,
			flagApp,
			stdcli.StringFlag("manifest", "m", "manifest file"),
			stdcli.StringFlag("generation", "g", "generation"),
			stdcli.BoolFlag("no-build", "", "skip build"),
			stdcli.BoolFlag("no-cache", "", "build withoit layer cache"),
			stdcli.BoolFlag("no-sync", "", "do not sync local changes into the running containers"),
			stdcli.IntFlag("shift", "s", "shift local port numbers (generation 1 only)"),
		},
		Usage: "[service] [service...]",
	})
}

func Start(rack sdk.Interface, c *stdcli.Context) error {
	opts := start.Options{}

	if len(c.Args) > 0 {
		opts.Services = c.Args
	}

	opts.App = app(c)
	opts.Build = !c.Bool("no-build")
	opts.Cache = !c.Bool("no-cache")
	opts.Sync = !c.Bool("no-sync")

	if v := c.String("manifest"); v != "" {
		opts.Manifest = v
	}

	if v := c.Int("shift"); v > 0 {
		opts.Shift = v
	}

	if c.String("generation") == "1" || c.LocalSetting("generation") == "1" || filepath.Base(opts.Manifest) == "docker-compose.yml" {
		if len(c.Args) >= 1 {
			opts.Services = []string{c.Arg(0)}
		}

		if len(c.Args) > 1 {
			opts.Command = c.Args[1:]
		}

		return Starter.Start1(opts)
	}

	if !localRackRunning(c) {
		return fmt.Errorf("local rack not found, try `sudo convox rack install local`")
	}

	var p structs.Provider

	if rack != nil {
		s, err := rack.SystemGet()
		if err != nil {
			return err
		}
		if s.Provider == "local" || s.Provider == "klocal" {
			p = rack
		}
	}

	if p == nil {
		r, err := matchRack(c, "local/")
		if err != nil {
			if strings.HasPrefix(err.Error(), "ambiguous rack name") {
				return fmt.Errorf("multiple local racks detected, use `convox switch` to select one")
			}
			return err
		}

		cl, err := sdk.New(fmt.Sprintf("https://rack.%s", strings.TrimPrefix(r.Name, "local/")))
		if err != nil {
			return err
		}

		p = cl
	}

	if p == nil {
		return fmt.Errorf("could not find local rack")
	}

	return Starter.Start2(p, opts)
}
