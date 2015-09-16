package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/convox/rack/cmd/convox/stdcli"
	"github.com/convox/release/version"
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
				Name:        "versions",
				Description: "list convox versions",
				Usage:       "",
				Action:      cmdVersions,
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

	specified := ""

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

func cmdVersions(c *cli.Context) {
	vs, err := version.All()

	if err != nil {
		return
	}

	var t *stdcli.Table

	if c.Bool("unpublished") {
		t = stdcli.NewTable("RELEASE", "PUBLISHED", "REQUIRED", "DESCRIPTION")
		for _, v := range vs {
			t.AddRow(v.Version, humanizeBool(v.Published), humanizeBool(v.Required), v.Description)
		}
	} else {
		t = stdcli.NewTable("RELEASE", "REQUIRED", "DESCRIPTION")
		for _, v := range vs {
			if v.Published {
				t.AddRow(v.Version, humanizeBool(v.Required), v.Description)
			}
		}
	}

	t.Print()
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
