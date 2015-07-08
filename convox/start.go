package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "start",
		Description: "start an app for local development",
		Usage:       "<directory>",
		Action:      cmdStart,
	})
}

func cmdStart(c *cli.Context) {
	wd := "."

	if len(c.Args()) > 0 {
		wd = c.Args()[0]
	}

	dir, app, err := stdcli.DirApp(c, wd)

	if err != nil {
		stdcli.Error(err)
		return
	}

	err = buildLocal(dir, app)

	if err != nil {
		stdcli.Error(err)
		return
	}

	err = stdcli.Run("docker-compose", "up")

	if err != nil {
		stdcli.Error(err)
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
