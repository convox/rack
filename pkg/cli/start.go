package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
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
	ctx, cancel := context.WithCancel(context.Background())

	go handleInterrupt(cancel)

	opts := start.Options{Output: c}

	opts.App = app(c)
	opts.Build = !c.Bool("no-build")
	opts.Cache = !c.Bool("no-cache")
	opts.Sync = !c.Bool("no-sync")

	if v := c.String("manifest"); v != "" {
		opts.Manifest = v
	}

	if c.String("generation") == "1" || c.LocalSetting("generation") == "1" || filepath.Base(opts.Manifest) == "docker-compose.yml" {
		opts1 := start.Options1{Options: opts}

		if len(c.Args) >= 1 {
			opts1.Service = c.Arg(0)
		}

		if len(c.Args) > 1 {
			opts1.Command = c.Args[1:]
		}

		if v := c.Int("shift"); v > 0 {
			opts1.Shift = v
		}

		return Starter.Start1(opts1)
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

	opts2 := start.Options2{Options: opts, Provider: p}

	if len(c.Args) > 0 {
		opts2.Services = c.Args
	}

	return Starter.Start2(ctx, opts2)
}

func handleInterrupt(cancel context.CancelFunc) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, os.Kill)
	<-ch
	fmt.Println("")
	cancel()
}
