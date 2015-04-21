package main

import (
	"encoding/json"
	"fmt"
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
	base := "."

	if len(c.Args()) > 0 {
		base = c.Args()[0]
	}

	base, err := filepath.Abs(base)

	if err != nil {
		panic(err)
	}

	if exists(filepath.Join(base, "Dockerfile")) {
		err := startDockerfile(base)

		if err != nil {
			panic(err)
		}
	}
}

func startDockerfile(base string) error {
	err := run("docker", "build", "-t", "convox-app", base)

	if err != nil {
		return err
	}

	data, err := query("docker", "inspect", "-f", "{{ json .ContainerConfig.ExposedPorts }}", "convox-app")

	if err != nil {
		return err
	}

	var ports map[string]interface{}

	err = json.Unmarshal(data, &ports)

	if err != nil {
		return err
	}

	args := []string{"run"}
	cur := 5000

	for port, _ := range ports {
		args = append(args, "-p")
		args = append(args, fmt.Sprintf("%d:%s", cur, strings.Split(port, "/")[0]))
		cur += 100
	}

	args = append(args, "convox-app")

	err = run("docker", args...)

	if err != nil {
		return err
	}

	return nil
}

func exists(filename string) bool {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false
	}

	return true
}

func query(bin string, args ...string) ([]byte, error) {
	return exec.Command(bin, args...).CombinedOutput()
}

func run(bin string, args ...string) error {
	cmd := exec.Command(bin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
