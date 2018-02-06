package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/convox/rack/cmd/convox/helpers"
	"github.com/convox/rack/cmd/convox/stdcli"
	"github.com/convox/rack/manifest1"
	"github.com/convox/rack/sdk"
	"github.com/convox/rack/structs"
	"github.com/fsouza/go-dockerclient"
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
				Name:   "file, f",
				EnvVar: "COMPOSE_FILE",
				Value:  "",
				Usage:  "path to manifest file",
			},
			cli.StringFlag{
				Name:  "generation, g",
				Usage: "generation of app",
			},
			cli.BoolFlag{
				Name:  "no-build",
				Usage: "dont run the build process",
			},
			cli.BoolFlag{
				Name:  "no-cache",
				Usage: "pull fresh image dependencies",
			},
			cli.BoolFlag{
				Name:  "no-sync",
				Usage: "do not synchronize local file changes into the running containers",
			},
			cli.IntFlag{
				Name:  "shift",
				Usage: "shift allocated port numbers by the given amount",
			},
		},
	})
}

func cmdStart(c *cli.Context) error {
	if err := dockerTest(); err != nil {
		return stdcli.Error(err)
	}

	opts := startOptions{}

	if len(c.Args()) > 0 {
		opts.Service = c.Args()[0]
	}

	if len(c.Args()) > 1 {
		opts.Command = c.Args()[1:]
	}

	opts.Build = !c.Bool("no-build")
	opts.Cache = !c.Bool("no-cache")
	opts.Sync = !c.Bool("no-sync")

	if v := c.String("file"); v != "" {
		opts.Config = v
	}

	if v := c.Int("shift"); v > 0 {
		opts.Shift = v
	}

	opts.Id, _ = currentId()

	if stdcli.ReadSetting("generation") == "2" || c.String("generation") == "2" || filepath.Base(opts.Config) == "convox.yml" {
		if err := startGeneration2(opts); err != nil {
			return stdcli.Error(err)
		}
	} else {
		if err := startGeneration1(opts); err != nil {
			return stdcli.Error(err)
		}
	}

	return nil
}

type startOptions struct {
	Build   bool
	Cache   bool
	Command []string
	Config  string
	Id      string
	Service string
	Shift   int
	Sync    bool
}

func startGeneration1(opts startOptions) error {
	opts.Config = helpers.Coalesce(opts.Config, "docker-compose.yml")

	if !helpers.Exists(opts.Config) {
		return fmt.Errorf("manifest not found: %s", opts.Config)
	}

	app, err := stdcli.DefaultApp(opts.Config)
	if err != nil {
		return err
	}

	m, err := manifest1.LoadFile(opts.Config)
	if err != nil {
		return err
	}

	if errs := m.Validate(); len(errs) > 0 {
		for _, e := range errs[1:] {
			stdcli.Error(e)
		}
		return errs[0]
	}

	if s := opts.Service; s != "" {
		if _, ok := m.Services[s]; !ok {
			return fmt.Errorf("service %s not found in manifest", s)
		}
	}

	if err := m.Shift(opts.Shift); err != nil {
		return err
	}

	// one-off commands don't need port validation
	if len(opts.Command) == 0 {
		pcc, err := m.PortConflicts()
		if err != nil {
			return err
		}
		if len(pcc) > 0 {
			return fmt.Errorf("ports in use: %v", pcc)
		}
	}

	r := m.Run(filepath.Dir(opts.Config), app, manifest1.RunOptions{
		Build:   opts.Build,
		Cache:   opts.Cache,
		Command: opts.Command,
		Service: opts.Service,
		Sync:    opts.Sync,
	})

	err = r.Start()
	if err != nil {
		r.Stop()
		return err
	}

	go handleInterrupt1(r)

	return r.Wait()
}

func startGeneration2(opts startOptions) error {
	app, err := stdcli.DefaultApp(opts.Config)
	if err != nil {
		return err
	}

	rack, err := sdk.New("https://localhost:5443")
	if err != nil {
		return err
	}

	if _, err := rack.AppGet(app); err != nil {
		return err
	}

	tar, err := createTarball(".")
	if err != nil {
		return err
	}

	o, err := rack.ObjectStore(app, "", bytes.NewReader(tar), structs.ObjectStoreOptions{})
	if err != nil {
		return err
	}

	b, err := rack.BuildCreate(app, "tgz", o.Url, structs.BuildCreateOptions{})
	if err != nil {
		return err
	}

	if err := waitForBuildGeneration2(rack, app, b.Id); err != nil {
		return err
	}

	logs, err := rack.BuildLogs(app, b.Id, structs.LogsOptions{})
	if err != nil {
		return err
	}

	if _, err := io.Copy(os.Stdout, logs); err != nil {
		return err
	}

	b, err = rack.BuildGet(app, b.Id)
	if err != nil {
		return err
	}

	switch b.Status {
	case "created", "running", "complete":
	case "failed":
		return fmt.Errorf("build failed")
	default:
		return fmt.Errorf("unknown build status: %s", b.Status)
	}

	if err := rack.ReleasePromote(app, b.Release); err != nil {
		return err
	}

	r, err := rack.ReleaseGet(app, b.Release)
	if err != nil {
		return err
	}

	switch r.Status {
	case "created", "promoting", "promoted", "active":
	case "failed":
		return fmt.Errorf("release failed")
	default:
		return fmt.Errorf("unknown release status: %s", r.Status)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	// wd, err := os.Getwd()
	// if err != nil {
	//   return err
	// }

	// for _, s := range m.Services {
	//   go watchChanges(Rack(c), wd, m, app, s.Name, errch)
	// }

	logs, err = rack.AppLogs(app, structs.LogsOptions{Follow: true, Prefix: true})
	if err != nil {
		return err
	}

	if _, err := io.Copy(os.Stdout, logs); err != nil {
		return err
	}

	return nil
}

// func startGeneration2(opts startOptions) error {
//   opts.Config = helpers.Coalesce(opts.Config, "convox.yml")

//   if !helpers.Exists(opts.Config) {
//     return fmt.Errorf("manifest not found: %s", opts.Config)
//   }

//   app, err := stdcli.DefaultApp(opts.Config)
//   if err != nil {
//     return err
//   }

//   data, err := ioutil.ReadFile(opts.Config)
//   if err != nil {
//     return err
//   }

//   dir := filepath.Dir(app)

//   env := structs.Environment{}

//   if data, err := ioutil.ReadFile(filepath.Join(dir, ".env")); err == nil {
//     if err := env.Load(data); err != nil {
//       return err
//     }
//   }

//   m, err := manifest.Load(data, manifest.Environment(env))
//   if err != nil {
//     return err
//   }

//   bopts := manifest.BuildOptions{
//     Development: true,
//     Stdout:      m.Writer("build", os.Stdout),
//     Stderr:      m.Writer("build", os.Stderr),
//   }

//   if err := m.Build(app, "latest", bopts); err != nil {
//     return err
//   }

//   ropts := manifest.RunOptions{
//     Bind:   true,
//     Env:    env,
//     Stdout: os.Stdout,
//     Stderr: os.Stderr,
//   }

//   if err := m.Run(app, ropts); err != nil {
//     return err
//   }

//   return nil
// }

func handleInterrupt1(run manifest1.Run) {
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
	if err != nil {
		return err
	}
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

func waitForBuildGeneration2(rack *sdk.Client, app, id string) error {
	for {
		time.Sleep(1 * time.Second)

		b, err := rack.BuildGet(app, id)
		if err != nil {
			return err
		}

		switch b.Status {
		case "created":
			break
		case "running", "complete":
			return nil
		case "failed":
			return fmt.Errorf("build failed")
		default:
			return fmt.Errorf("unknown build status: %s", b.Status)
		}
	}
}
