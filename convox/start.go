package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/manifest"
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

	m, err := manifest.Generate(dir)

	if err != nil {
		stdcli.Error(err)
		return
	}

	missing := m.MissingEnvironment()

	if len(missing) > 0 {
		stdcli.Error(fmt.Errorf("env expected: %s", strings.Join(missing, ", ")))
		return
	}

	wanted, err := m.PortsWanted()

	if err != nil {
		stdcli.Error(err)
		return
	}

	conflicts := make([]string, 0)

	host := "127.0.0.1"

	if h := os.Getenv("DOCKER_HOST"); h != "" {
		u, err := url.Parse(h)

		if err != nil {
			stdcli.Error(err)
			return
		}

		parts := strings.Split(u.Host, ":")
		host = parts[0]
	}

	for _, p := range wanted {
		conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", host, p))

		if err == nil {
			conflicts = append(conflicts, p)
			defer conn.Close()
		}
	}

	if len(conflicts) > 0 {
		stdcli.Error(fmt.Errorf("ports in use: %s", strings.Join(conflicts, ", ")))
		return
	}

	cwd, err := os.Getwd()

	if err != nil {
		stdcli.Error(err)
		return
	}

	errors := m.Build(app, cwd)

	if len(errors) != 0 {
		fmt.Printf("errors: %+v\n", errors)
		return
	}

	errors = m.Run(app)

	if len(errors) != 0 {
		fmt.Printf("errors: %+v\n", errors)
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
