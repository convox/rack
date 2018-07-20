package main

import (
	"fmt"
	"path/filepath"

	"github.com/convox/rack/sdk"
	"github.com/convox/rack/start"
	"github.com/convox/stdcli"
)

func init() {
	CLI.Command("start", "start an application for local development", Start, stdcli.CommandOptions{
		Flags: []stdcli.Flag{
			flagRack,
			flagApp,
			flagId,
			stdcli.StringFlag("file", "f", "manifest file"),
			stdcli.StringFlag("generation", "g", "generation"),
			stdcli.BoolFlag("no-build", "", "skip build"),
			stdcli.BoolFlag("no-cache", "", "build withoit layer cache"),
			stdcli.BoolFlag("no-sync", "", "do not sync local changes into the running containers"),
			stdcli.IntFlag("shift", "s", "shift local port numbers (generation 1 only)"),
		},
		Usage: "[service] [command]",
	})
}

func Start(c *stdcli.Context) error {
	opts := start.Options{}

	if len(c.Args) > 0 {
		opts.Services = c.Args
	}

	if len(c.Args) > 1 {
		opts.Command = c.Args[1:]
	}

	opts.App = app(c)
	opts.Build = !c.Bool("no-build")
	opts.Cache = !c.Bool("no-cache")
	opts.Sync = !c.Bool("no-sync")

	if v := c.String("file"); v != "" {
		opts.Manifest = v
	}

	if v := c.Int("shift"); v > 0 {
		opts.Shift = v
	}

	if c.String("generation") == "1" || c.LocalSetting("generation") == "1" || filepath.Base(opts.Manifest) == "docker-compose.yml" {
		return start.Start1(opts)
	}

	if !localRackRunning() {
		return fmt.Errorf("local rack not found, try `sudo convox rack install local`")
	}

	host, err := currentHost(c)
	if err != nil {
		return err
	}

	r := currentRack(c, host)

	if _, err := currentEndpoint(c, r); err == nil {
		p := provider(c)

		s, err := p.SystemGet()
		if err == nil && s.Provider == "local" {
			opts.Provider = p
		}
	}

	if opts.Provider == nil {
		r, err := matchRack(c, "local")
		if err != nil {
			return err
		}

		cl, err := sdk.New(fmt.Sprintf("https://rack.%s", r.Name))
		if err != nil {
			return err
		}

		opts.Provider = cl
	}

	return start.Start2(opts)
}
