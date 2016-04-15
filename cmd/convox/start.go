package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
			stdcli.Error(err)
			return
		}

		fmt.Printf("Generated: %s\n", strings.Join(changes, ", "))

		m, err = manifest.Read(dir, file)

		if err != nil {
			stdcli.Error(err)
			return
		}
	}

	conflicts, err := m.PortConflicts()

	if err != nil {
		stdcli.Error(err)
		return
	}

	if len(conflicts) > 0 {
		stdcli.Error(fmt.Errorf("ports in use: %s", strings.Join(conflicts, ", ")))
		return
	}

	missing, err := m.MissingEnvironment(cache)

	if err != nil {
		stdcli.Error(err)
		return
	}

	if len(missing) > 0 {
		stdcli.Error(fmt.Errorf("env expected: %s", strings.Join(missing, ", ")))
		return
	}

	errors := m.Build(app, dir, cache)

	if len(errors) != 0 {
		fmt.Printf("errors: %+v\n", errors)
		return
	}

	ch := make(chan []error)

	go func() {
		ch <- m.Run(app, cache)
	}()

	if c.Bool("sync") && !stdcli.ReadSetting("sync") == "false" {
		m.Sync(app)
	}

	errors = <-ch

	if len(errors) != 0 {
		// TODO figure out what to do here
		// fmt.Printf("errors: %+v\n", errors)
		return
	}
}

func buildLocal(dir, app string) error {
	abs, err := filepath.Abs(dir)

	if err != nil {
		return err
	}

	err = run("docker", "--tlsverify=false", "run", "-i", "-v", "/var/run/docker.sock:/var/run/docker.sock", "-v", fmt.Sprintf("%s:/source", abs), "convox/build", app, "/source")

	if err != nil {
		return err
	}

	return nil
}

func run(command string, args ...string) error {
	cmd := exec.Command(command, args...)

	stdout, err := cmd.StdoutPipe()

	if err != nil {
		return err
	}

	stderr, err := cmd.StderrPipe()

	if err != nil {
		return err
	}

	cmd.Start()

	scanner := bufio.NewScanner(stdout)

	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), "|", 2)

		if len(parts) == 2 {
			switch parts[0] {
			case "build", "compose":
				fmt.Println(parts[1])
			case "manifest":
			default:
				fmt.Println(scanner.Text())
			}
		}
	}

	s, err := ioutil.ReadAll(stderr)

	if err != nil {
		return err
	}

	err = cmd.Wait()

	if stdcli.Debug() {
		fmt.Fprintf(os.Stderr, "DEBUG: exec: '%v', '%v', '%v', '%v'\n", command, args, err, string(s))
	}

	if err != nil {
		return err
	}

	return nil
}

func cmdInit(c *cli.Context) {
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
		stdcli.Error(err)
		return
	}

	if len(changed) > 0 {
		fmt.Printf("Generated: %s\n", strings.Join(changed, ", "))
	}
}
