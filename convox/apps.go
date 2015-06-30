package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
)

type App struct {
	Name string

	Status     string
	Repository string

	Outputs    map[string]string
	Parameters map[string]string
	Tags       map[string]string
}

type Apps []App

func (a App) PrintInfo() {
	var ps sort.StringSlice = make([]string, 0)
	ports := make(map[string]string)

	for k := range a.Parameters {
		if strings.HasSuffix(k, "Command") {
			ps = append(ps, strings.TrimSuffix(k, "Command"))
		}

		if strings.HasSuffix(k, "Balancer") {
			p := strings.Split(k, "Port")[0]
			ports[p] = a.Parameters[k]
		}
	}

	ps.Sort()

	fmt.Printf("%-12v %v\n", "Name", a.Name)
	fmt.Printf("%-12v %v\n", "Status", a.Status)
	fmt.Printf("%-12v %v\n", "Release", a.Parameters["Release"])

	for _, p := range ps {
		cmd := a.Parameters[p+"Command"]
		port := ports[p]

		if cmd != "" {
			fmt.Printf("%-12v %v\n", p, a.Parameters[p+"Command"])
		} else {
			fmt.Printf("%-12v [image]\n", p)
		}

		if port != "" {
			fmt.Printf("%-12v %s:%s\n", p+" Host", a.Outputs["BalancerHost"], port)
		}
	}
}

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "apps",
		Action:      cmdApps,
		Description: "list apps",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "app",
				Usage: "app name. If not specified, use current directory.",
			},
		},
		Subcommands: []cli.Command{
			{
				Name:   "create",
				Usage:  "convox apps create [name]",
				Action: cmdAppCreate,
			},
		},
	})
}

func cmdApps(c *cli.Context) {
	data, err := ConvoxGet("/apps")

	if err != nil {
		stdcli.Error(err)
		return
	}

	var apps *Apps
	err = json.Unmarshal(data, &apps)

	if err != nil {
		stdcli.Error(err)
		return
	}

	for _, app := range *apps {
		fmt.Printf("%s\n", app.Name)
	}
}

func cmdAppCreate(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	if c.Args()[0] != "" {
		app = c.Args()[0]
	}

	v := url.Values{}
	v.Set("name", app)
	data, err := ConvoxPostForm("/apps", v)

	if err != nil {
		stdcli.Error(err)
		return
	}

	data, err = ConvoxGet("/apps/" + app)

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

	fmt.Printf("Created %s.\n", a.Name)
}
