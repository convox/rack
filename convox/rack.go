package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "rack",
		Description: "manage your Convox rack",
		Usage:       "",
		Action:      cmdRack,
		Subcommands: []cli.Command{
			{
				Name:        "update",
				Description: "update rack to the latest version",
				Usage:       "[version]",
				Action:      cmdRackUpdate,
			},
			{
				Name:        "scale",
				Description: "scale the rack capacity",
				Usage:       "",
				Action:      cmdRackScale,
				Flags: []cli.Flag{
					cli.IntFlag{
						Name:  "count",
						Usage: "horizontally scale the instance count, e.g. 3 or 10",
					},
					cli.StringFlag{
						Name:  "type",
						Usage: "vertically scale the instance type, e.g. t2.small or c3.xlargs",
					},
				},
			},
		},
	})
}

func cmdRack(c *cli.Context) {
	system, err := rackClient().GetSystem()

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Printf("Name     %s\n", system.Name)
	fmt.Printf("Status   %s\n", system.Status)
	fmt.Printf("Version  %s\n", system.Version)
	fmt.Printf("Count    %d\n", system.Count)
	fmt.Printf("Type     %s\n", system.Type)
}

func cmdRackUpdate(c *cli.Context) {
	version := ""

	if len(c.Args()) == 0 {
		v, err := latestVersion()

		if err != nil {
			stdcli.Error(err)
			return
		}

		version = v
	} else {
		version = c.Args()[0]
	}

	system, err := rackClient().UpdateSystem(version)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Printf("Name     %s\n", system.Name)
	fmt.Printf("Status   %s\n", system.Status)
	fmt.Printf("Version  %s\n", system.Version)
	fmt.Printf("Count    %d\n", system.Count)
	fmt.Printf("Type     %s\n", system.Type)
}

func cmdRackScale(c *cli.Context) {
	count := 0
	typ := ""

	if c.IsSet("count") {
		count = c.Int("count")
	}

	if c.IsSet("type") {
		typ = c.String("type")
	}

	system, err := rackClient().ScaleSystem(count, typ)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Printf("Name     %s\n", system.Name)
	fmt.Printf("Status   %s\n", system.Status)
	fmt.Printf("Version  %s\n", system.Version)
	fmt.Printf("Count    %d\n", system.Count)
	fmt.Printf("Type     %s\n", system.Type)
}

func latestVersion() (string, error) {
	resp, err := http.Get("http://convox.s3.amazonaws.com/release/latest/version")

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(body)), nil
}
