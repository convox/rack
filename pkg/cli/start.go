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

	if c.String("generation") == "1" || c.LocalSetting("generation") == "1" || filepath.Base(c.String("manifest")) == "docker-compose.yml" {
		opts := start.Options1{
			App:      app(c),
			Build:    !c.Bool("no-build"),
			Cache:    !c.Bool("no-cache"),
			Manifest: c.String("manifest"),
			Shift:    c.Int("shift"),
			Sync:     !c.Bool("no-sync"),
		}

		if len(c.Args) >= 1 {
			opts.Service = c.Arg(0)
		}

		if len(c.Args) > 1 {
			opts.Command = c.Args[1:]
		}

		return Starter.Start1(ctx, opts)
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
		if s.Provider == "local" || s.Provider == "klocal" || s.Provider == "kaws" {
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

	opts := start.Options2{
		App:      app(c),
		Build:    !c.Bool("no-build"),
		Cache:    !c.Bool("no-cache"),
		Manifest: c.String("manifest"),
		Provider: p,
		Sync:     !c.Bool("no-sync"),
	}

	if len(c.Args) > 0 {
		opts.Services = c.Args
	}

	return Starter.Start2(ctx, c, opts)
}

func handleInterrupt(cancel context.CancelFunc) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, os.Kill)
	<-ch
	fmt.Println("")
	cancel()
}
