package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/codegangsta/cli"
	"github.com/convox/rack/api/manifest"
	"github.com/convox/rack/cmd/convox/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "start",
		Description: "start an app for local development",
		Usage:       "[directory]",
		Action:      cmdStart,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "file, f",
				Value: "docker-compose.yml",
				Usage: "path to an alternate docker compose manifest file",
			},
			cli.BoolFlag{
				Name:  "no-cache",
				Usage: "pull fresh image dependencies",
			},
			cli.BoolTFlag{
				Name:  "sync",
				Usage: "synchronize local file changes into the running containers",
			},
		},
	})
	stdcli.RegisterCommand(cli.Command{
		Name:        "init",
		Description: "initialize an app for local development",
		Usage:       "[directory]",
		Action:      cmdInit,
	})
}

func cmdStart(c *cli.Context) {
	ep := stdcli.EventProperties{Start: time.Now()}

	distinctId, err := currentId()
	if err != nil {
		stdcli.EventSend("cli-start", distinctId, stdcli.EventProperties{Error: err})
	}

	cache := !c.Bool("no-cache")

	wd := "."

	if len(c.Args()) > 0 {
		wd = c.Args()[0]
	}

	dir, app, err := stdcli.DirApp(c, wd)
	if err != nil {
		stdcli.Error(err)
		return
	}

	file := c.String("file")

	m, err := manifest.Read(dir, file)
	if err != nil {
		changes, err := manifest.Init(dir)
		if err != nil {
			stdcli.EventSend("cli-start", distinctId, stdcli.EventProperties{Error: err})
		}

		fmt.Printf("Generated: %s\n", strings.Join(changes, ", "))

		m, err = manifest.Read(dir, file)
		if err != nil {
			stdcli.EventSend("cli-start", distinctId, stdcli.EventProperties{Error: err})
		}
	}

	conflicts, err := m.PortConflicts()
	if err != nil {
		stdcli.EventSend("cli-start", distinctId, stdcli.EventProperties{Error: err})
	}

	if len(conflicts) > 0 {
		stdcli.Error(fmt.Errorf("ports in use: %s", strings.Join(conflicts, ", ")))
		return
	}

	missing, err := m.MissingEnvironment(cache, app)
	if err != nil {
		stdcli.EventSend("cli-start", distinctId, stdcli.EventProperties{Error: err})
	}

	if len(missing) > 0 {
		stdcli.Error(fmt.Errorf("env expected: %s", strings.Join(missing, ", ")))
		return
	}

	errors := m.Build(app, dir, cache)
	if len(errors) != 0 {
		stdcli.EventSend("cli-start", distinctId, stdcli.EventProperties{Error: errors[0]})
	}

	ch := make(chan []error)

	go func() {
		ch <- m.Run(app, cache)
	}()

	if c.Bool("sync") && stdcli.ReadSetting("sync") != "false" {
		m.Sync(app)
	}

	<-ch

	stdcli.EventSend("cli-start", distinctId, ep)
}

func cmdInit(c *cli.Context) {
	ep := stdcli.EventProperties{Start: time.Now()}

	distinctId, err := currentId()
	if err != nil {
		stdcli.EventSend("cli-init", distinctId, stdcli.EventProperties{Error: err})
	}

	wd := "."

	if len(c.Args()) > 0 {
		wd = c.Args()[0]
	}

	dir, _, err := stdcli.DirApp(c, wd)
	if err != nil {
		stdcli.Error(err)
		return
	}

	changed, err := manifest.Init(dir)
	if err != nil {
		stdcli.EventSend("cli-init", distinctId, stdcli.EventProperties{Error: err})
	}

	if len(changed) > 0 {
		fmt.Printf("Generated: %s\n", strings.Join(changed, ", "))
	}

	stdcli.EventSend("cli-init", distinctId, ep)
}
