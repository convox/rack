package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "info",
		Description: "see info about an app",
		Usage:       "[--app name]",
		Action:      cmdInfo,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "app",
				Usage: "app name. Inferred from current directory if not specified.",
			},
		},
	})
}

func cmdInfo(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	data, err := ConvoxGet("/apps/" + app)

	if err != nil {
		stdcli.Error(err)
		return
	}

	var a *App
	err = json.Unmarshal(data, &a)

	if err != nil {
		stdcli.Error(err)
		return
	}

	matcher := regexp.MustCompile(`^(\w+)Port\d+Balancer`)

	ports := []string{}

	for key, value := range a.Outputs {
		if m := matcher.FindStringSubmatch(key); m != nil {
			ports = append(ports, fmt.Sprintf("%s:%s", strings.ToLower(m[1]), value))
		}
	}

	processes := []string{}

	for key, _ := range a.Parameters {
		if strings.HasSuffix(key, "Image") {
			processes = append(processes, strings.ToLower(key[0:len(key)-5]))
		}
	}

	sort.Strings(processes)

	if len(processes) == 0 {
		processes = append(processes, "(none)")
	}

	if len(ports) == 0 {
		ports = append(ports, "(none)")
	}

	release := a.Parameters["Release"]

	if release == "" {
		release = "(none)"
	}

	fmt.Printf("Name       %s\n", a.Name)
	fmt.Printf("Status     %s\n", a.Status)
	fmt.Printf("Release    %s\n", release)
	fmt.Printf("Processes  %s\n", strings.Join(processes, " "))

	if a.Outputs["BalancerHost"] != "" {
		fmt.Printf("Hostname   %s\n", a.Outputs["BalancerHost"])
		fmt.Printf("Ports      %s\n", strings.Join(ports, " "))
	}
}
