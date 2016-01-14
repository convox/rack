package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/convox/release/version"
	"github.com/convox/rack/cmd/convox/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "rack",
		Description: "manage your Convox rack",
		Usage:       "",
		Action:      cmdRack,
		Subcommands: []cli.Command{
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
			{
				Name:        "update",
				Description: "update rack to the given version",
				Usage:       "[version]",
				Action:      cmdRackUpdate,
			},
			{
				Name:        "releases",
				Description: "list rack releases",
				Usage:       "",
				Action:      cmdRackReleases,
				Flags: []cli.Flag{
					cli.BoolFlag{
						Name:  "unpublished",
						Usage: "include unpublished versions",
					},
				},
			},
		},
	})
}

func cmdRack(c *cli.Context) {
	system, err := rackClient(c).GetSystem()

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
	versions, err := version.All()

	if err != nil {
		stdcli.Error(err)
		return
	}

	specified := "stable"

	if len(c.Args()) > 0 {
		specified = c.Args()[0]
	}

	version, err := versions.Resolve(specified)

	if err != nil {
		stdcli.Error(err)
		return
	}

	system, err := rackClient(c).UpdateSystem(version.Version)

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

	system, err := rackClient(c).ScaleSystem(count, typ)

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

func cmdRackReleases(c *cli.Context) {
	vs, err := version.All()

	if err != nil {
		return
	}

	selected := version.Versions{}

	for _, v := range vs {
		switch {
		case !v.Published && c.Bool("unpublished"):
			selected = append(selected, v)
		case v.Published:
			selected = append(selected, v)
		}
	}

	sort.Sort(sort.Reverse(selected))

	if len(selected) > 20 {
		selected = selected[0:20]
	}

	for _, v := range selected {
		fmt.Println(v.Version)
	}
}

func latestVersion() (string, error) {
	resp, err := http.Get("https://convox.s3.amazonaws.com/release/latest/version")

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
