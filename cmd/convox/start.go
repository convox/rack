package main

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/convox/rack/api/models"
	"github.com/convox/rack/cmd/convox/stdcli"
	"github.com/convox/rack/manifest"
	"github.com/fsouza/go-dockerclient"
	"github.com/robfig/cron"
	"gopkg.in/urfave/cli.v1"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "start",
		Description: "start an app for local development",
		Usage:       "[service] [command]",
		Action:      cmdStart,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "file, f",
				Value: "docker-compose.yml",
				Usage: "path to an alternate docker compose manifest file",
			},
			cli.BoolFlag{
				Name:  "no-build",
				Usage: "do not build the app images",
			},
			cli.BoolFlag{
				Name:  "no-cache",
				Usage: "pull fresh image dependencies",
			},
			cli.IntFlag{
				Name:  "shift",
				Usage: "shift allocated port numbers by the given amount",
			},
			cli.BoolFlag{
				Name:  "no-sync",
				Usage: "do not synchronize local file changes into the running containers",
			},
			cli.IntFlag{
				Name:   "start-wait",
				EnvVar: "CONVOX_START_WAIT",
				Value:  10,
				Usage:  "wait for containers to finish promoting before failing",
			},
		},
	})
}

func cmdStart(c *cli.Context) error {
	// go handleResize()
	var service string
	var command []string

	if len(c.Args()) > 0 {
		service = c.Args()[0]
	}

	if len(c.Args()) > 1 {
		command = c.Args()[1:]
	}

	id, err := currentId()
	if err != nil {
		stdcli.QOSEventSend("cli-start", id, stdcli.QOSEventProperties{Error: err})
	}

	err = dockerTest()
	if err != nil {
		stdcli.QOSEventSend("cli-start", id, stdcli.QOSEventProperties{ValidationError: err})
		return stdcli.Error(err)
	}

	dir, app, err := stdcli.DirApp(c, filepath.Dir(c.String("file")))
	if err != nil {
		stdcli.QOSEventSend("cli-start", id, stdcli.QOSEventProperties{ValidationError: err})
		return stdcli.Error(err)
	}

	appType := detectApplication(dir)
	m, err := manifest.LoadFile(c.String("file"))
	if err != nil {
		return stdcli.Error(err)
	}

	errs := m.Validate()
	if len(errs) > 0 {
		for _, e := range errs[1:] {
			stdcli.Error(e)
		}
		return stdcli.Error(errs[0])
	}

	if service != "" {
		_, ok := m.Services[service]
		if !ok {
			return stdcli.Error(fmt.Errorf("Service %s not found in manifest", service))
		}
	}

	if err := m.Shift(c.Int("shift")); err != nil {
		return stdcli.Error(err)
	}

	// one-off commands don't need port validation
	if len(command) == 0 {
		if pcc, err := m.PortConflicts(); err != nil || len(pcc) > 0 {
			if err == nil {
				err = fmt.Errorf("ports in use: %v", pcc)
			}
			stdcli.QOSEventSend("cli-start", id, stdcli.QOSEventProperties{
				ValidationError: err,
				AppType:         appType,
			})
			return stdcli.Error(err)
		}
	}

	r := m.Run(dir, app, manifest.RunOptions{
		Build:   !c.Bool("no-build"),
		Cache:   !c.Bool("no-cache"),
		Command: command,
		Service: service,
		Sync:    !c.Bool("no-sync"),
		Wait:    c.Int("start-wait"),
	})

	err = r.Start()
	if err != nil {
		r.Stop()

		stdcli.QOSEventSend("cli-start", id, stdcli.QOSEventProperties{
			ValidationError: err,
			AppType:         appType,
		})
		return stdcli.Error(err)
	}

	stdcli.QOSEventSend("cli-start", id, stdcli.QOSEventProperties{
		AppType: appType,
	})

	// Setup the local "cron jobs"
	for _, entry := range m.Services {
		labels := entry.LabelsByPrefix("convox.cron")
		processName := fmt.Sprintf("%s-%s", app, entry.Name)
		c := cron.New()

		for key, value := range labels {
			p, ok := r.Processes[processName]
			if !ok {
				continue
			}
			cronjob := models.NewCronJobFromLabel(key, value)

			rs := strings.NewReplacer("cron(", "0 ", ")", "")                 // cron pkg first field is in seconds so set to 0
			trigger := strings.TrimSuffix(rs.Replace(cronjob.Schedule), " *") // and doesn't recognize year so we trim it

			c.AddFunc(trigger, func() {
				cronProccesName := fmt.Sprintf("cron-%s-%04d", cronjob.Name, rand.Intn(9000))
				// Replace args with cron specific ones
				cronArgs := p.GenerateArgs(&manifest.ArgOptions{
					Command:     cronjob.Command,
					IgnorePorts: true,
					Name:        cronProccesName,
				})

				done := make(chan error)
				manifest.RunAsync(
					r.Output.Stream(cronProccesName),
					manifest.Docker(append([]string{"run"}, cronArgs...)...),
					done,
				)

				err := <-done
				if err != nil {
					fmt.Printf("error running %s: %s\n", cronProccesName, err.Error())
				}
			})
		}

		c.Start()
	}

	go handleInterrupt(r)

	return r.Wait()
}

func handleInterrupt(run manifest.Run) {
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
		return errors.New("could not connect to docker daemon, is it installed and running?")
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
