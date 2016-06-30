package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/convox/rack/cmd/convox/stdcli"
	"github.com/convox/rack/manifest"
	"github.com/fsouza/go-dockerclient"
	"gopkg.in/urfave/cli.v1"
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
	// go handleResize()

	id, err := currentId()
	if err != nil {
		stdcli.QOSEventSend("cli-start", id, stdcli.QOSEventProperties{Error: err})
	}

	err = dockerTest()
	if err != nil {
		stdcli.QOSEventSend("cli-start", id, stdcli.QOSEventProperties{ValidationError: err})
	}

	dir, app, err := stdcli.DirApp(c, filepath.Dir(c.String("file")))
	if err != nil {
		stdcli.QOSEventSend("cli-start", id, stdcli.QOSEventProperties{ValidationError: err})
	}

	appType := detectApplication(dir)

	m, err := manifest.LoadFile(c.String("file"))
	if err != nil {
		return stdcli.ExitError(err)
	}

	if shift := c.Int("shift"); shift > 0 {
		m.Shift(shift)
	}

	if pcc, err := m.PortConflicts(); err != nil || len(pcc) > 0 {
		if err == nil {
			err = fmt.Errorf("ports in use: %v", pcc)
		}
		stdcli.QOSEventSend("cli-start", id, stdcli.QOSEventProperties{
			ValidationError: err,
			AppType:         appType,
		})
	}

	r := m.Run(dir, app)

	if err := r.Start(); err != nil {
		stdcli.QOSEventSend("cli-start", id, stdcli.QOSEventProperties{
			ValidationError: err,
			AppType:         appType,
		})
	}
	stdcli.QOSEventSend("cli-start", id, stdcli.QOSEventProperties{
		AppType: appType,
	})

	go handleInterrupt(r, id)

	return r.Wait()
}

func handleInterrupt(run manifest.Run, id string) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, os.Kill)
	<-ch
	fmt.Println("")
	run.Stop()
	os.Exit(0)
}

func dockerTest() error {
	dockerTest := exec.Command("docker", "images")
	err := dockerTest.Run()
	if err != nil {
		return stdcli.ExitError(errors.New("could not connect to docker daemon, is it installed and running?"))
	}

	dockerVersionTest, err := docker.NewClientFromEnv()
	if err != nil {
		return err
	}

	minDockerVersion, err := docker.NewAPIVersion("1.9")
	e, err := dockerVersionTest.Version()
	if err != nil {
		return err
	}

	currentVersionParts := strings.Split(e.Get("Version"), ".")
	currentVersion, err := docker.NewAPIVersion(fmt.Sprintf("%s.%s", currentVersionParts[0], currentVersionParts[1]))
	if err != nil {
		return err
	}

	if !(currentVersion.GreaterThanOrEqualTo(minDockerVersion)) {
		return errors.New("Your version of docker is out of date (min: 1.9)")
	}
	return nil
}
