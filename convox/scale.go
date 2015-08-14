package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "scale",
		Description: "scale an app's processes",
		Usage:       "PROCESS [--count 2] [--memory 512]",
		Action:      cmdScale,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "app",
				Usage: "App name. Inferred from current directory if not specified.",
			},
			cli.IntFlag{
				Name:  "count",
				Value: 1,
				Usage: "Number of processes to keep running for specified process type.",
			},
			cli.IntFlag{
				Name:  "memory",
				Value: 256,
				Usage: "Amount of memory, in MB, available to specified process type.",
			},
		},
	})
}

func cmdScale(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	if len(c.Args()) > 1 {
		stdcli.Usage(c, "scale")
		return
	}

	v := url.Values{}

	if c.IsSet("count") {
		v.Set("count", c.String("count"))
	}

	if c.IsSet("memory") {
		v.Set("mem", c.String("memory"))
	}

	if len(v) > 0 {
		_, err = ConvoxPostForm("/apps/"+app, v)

		if err != nil {
			stdcli.Error(err)
			return
		}
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

	processes := make(map[string]Process, 0)

	longest := 7

	for k, v := range a.Parameters {
		if !strings.HasSuffix(k, "DesiredCount") {
			continue
		}

		ps := strings.ToLower(strings.Replace(k, "DesiredCount", "", 1))

		i, err := strconv.Atoi(v)

		if err != nil {
			stdcli.Error(err)
			return
		}

		processes[ps] = Process{Name: ps, Count: i}
	}

	fmt.Printf(fmt.Sprintf("%%-%ds  %%-5s  %%-5s\n", longest), "PROCESS", "COUNT", "MEM")

	for _, ps := range processes {
		fmt.Printf(fmt.Sprintf("%%-%ds  %%-5d  %%-5s\n", longest), ps.Name, ps.Count, a.Parameters["Memory"])
	}
}
