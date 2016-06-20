package main

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"gopkg.in/urfave/cli.v1"

	"github.com/convox/rack/api/manifest"
	"github.com/convox/rack/cmd/convox/stdcli"
	"github.com/fsouza/go-dockerclient"
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

func cmdStart(c *cli.Context) error {
	ep := stdcli.QOSEventProperties{Start: time.Now()}

	distinctId, err := currentId()
	if err != nil {
		return stdcli.QOSEventSend("cli-start", distinctId, stdcli.QOSEventProperties{Error: err})
	}

	dockerTest := exec.Command("docker", "images")
	err = dockerTest.Run()
	if err != nil {
		return stdcli.ExitError(errors.New("could not connect to docker daemon, is it installed and running?"))
	}

	dockerVersionTest, err := docker.NewClientFromEnv()
	if err != nil {
		return stdcli.ExitError(err)
	}

	minDockerVersion, err := docker.NewAPIVersion("1.9")
	e, err := dockerVersionTest.Version()
	if err != nil {
		return stdcli.ExitError(err)
	}

	currentVersionParts := strings.Split(e.Get("Version"), ".")
	currentVersion, err := docker.NewAPIVersion(fmt.Sprintf("%s.%s", currentVersionParts[0], currentVersionParts[1]))
	if err != nil {
		return stdcli.ExitError(err)
	}

	if !(currentVersion.GreaterThanOrEqualTo(minDockerVersion)) {
		return stdcli.ExitError(errors.New("Your version of docker is out of date (min: 1.9)"))
	}

	cache := !c.Bool("no-cache")

	shift := 0

	if si := c.Int("shift"); si > 0 {
		shift = si
	}

	wd := "."

	if len(c.Args()) > 0 {
		wd = c.Args()[0]
	}

	dir, app, err := stdcli.DirApp(c, wd)
	if err != nil {
		return stdcli.ExitError(err)
	}

	file := c.String("file")

	m, err := manifest.Read(dir, file)
	if err != nil {
		switch err.(type) {
		case *manifest.YAMLError:
			return stdcli.ExitError(fmt.Errorf(
				"Invalid manifest (%s): %s", file, err.Error(),
			))
		default:
			err := manifest.Init(dir)
			if err != nil {
				return stdcli.QOSEventSend("cli-start", distinctId, stdcli.QOSEventProperties{Error: err})
			}

			m, err = manifest.Read(dir, file)
			if err != nil {
				return stdcli.QOSEventSend("cli-start", distinctId, stdcli.QOSEventProperties{Error: err})
			}
		}
	}

	conflicts, err := m.PortConflicts(shift)
	if err != nil {
		return stdcli.QOSEventSend("cli-start", distinctId, stdcli.QOSEventProperties{Error: err})
	}

	if len(conflicts) > 0 {
		return stdcli.ExitError(fmt.Errorf("ports in use: %s", strings.Join(conflicts, ", ")))
	}

	missing, err := m.MissingEnvironment(cache, app)
	if err != nil {
		stdcli.QOSEventSend("cli-start", distinctId, stdcli.QOSEventProperties{Error: err})
	}

	if len(missing) > 0 {
		return stdcli.ExitError(fmt.Errorf("env expected: %s", strings.Join(missing, ", ")))
	}

	errors := m.Build(app, dir, cache)
	if len(errors) != 0 {
		return stdcli.QOSEventSend("cli-start", distinctId, stdcli.QOSEventProperties{Error: errors[0]})
	}

	sync := c.Bool("sync") && (stdcli.ReadSetting("sync") != "false")

	ch := make(chan []error)

	go func() {
		ch <- m.Run(app, cache, sync, shift)
	}()

	<-ch

	return stdcli.QOSEventSend("cli-start", distinctId, ep)
}
