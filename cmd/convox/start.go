package main

import (
	"fmt"
	"strconv"
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
			cli.IntFlag{
				Name:  "shift",
				Usage: "Shift allocated port numbers by the given amount",
			},
			cli.BoolTFlag{
				Name:  "sync",
				Usage: "synchronize local file changes into the running containers",
			},
		},
	})
}

func cmdStart(c *cli.Context) {
	ep := stdcli.QOSEventProperties{Start: time.Now()}

	distinctId, err := currentId()
	if err != nil {
		stdcli.QOSEventSend("cli-start", distinctId, stdcli.QOSEventProperties{Error: err})
	}

	cache := !c.Bool("no-cache")

	shift := 0

	if ss := stdcli.ReadSetting("shift"); ss != "" {
		shift, err = strconv.Atoi(ss)

		if err != nil {
			stdcli.Error(fmt.Errorf(".convox/shift must contain a number"))
			return
		}
	}

	if si := c.Int("shift"); si > 0 {
		shift = si
	}

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
		err := manifest.Init(dir)

		if err != nil {
			stdcli.QOSEventSend("cli-start", distinctId, stdcli.QOSEventProperties{Error: err})
		}

		m, err = manifest.Read(dir, file)
		if err != nil {
			stdcli.QOSEventSend("cli-start", distinctId, stdcli.QOSEventProperties{Error: err})
		}
	}

	conflicts, err := m.PortConflicts(shift)
	if err != nil {
		stdcli.QOSEventSend("cli-start", distinctId, stdcli.QOSEventProperties{Error: err})
	}

	if len(conflicts) > 0 {
		stdcli.Error(fmt.Errorf("ports in use: %s", strings.Join(conflicts, ", ")))
		return
	}

	missing, err := m.MissingEnvironment(cache, app)
	if err != nil {
		stdcli.QOSEventSend("cli-start", distinctId, stdcli.QOSEventProperties{Error: err})
	}

	if len(missing) > 0 {
		stdcli.Error(fmt.Errorf("env expected: %s", strings.Join(missing, ", ")))
		return
	}

	errors := m.Build(app, dir, cache)
	if len(errors) != 0 {
		stdcli.QOSEventSend("cli-start", distinctId, stdcli.QOSEventProperties{Error: errors[0]})
	}

	ch := make(chan []error)

	go func() {
		ch <- m.Run(app, cache, shift)
	}()

	if c.Bool("sync") && stdcli.ReadSetting("sync") != "false" {
		m.Sync(app)
	}

	<-ch

	stdcli.QOSEventSend("cli-start", distinctId, ep)
}
