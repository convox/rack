package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/convox/rack/cmd/convox/stdcli"
	"github.com/convox/version"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "rack",
		Description: "manage your Convox rack",
		Usage:       "",
		Action:      cmdRack,
		Subcommands: []cli.Command{
			{
				Name:        "params",
				Description: "list advanced rack parameters",
				Usage:       "",
				Action:      cmdRackParams,
				Subcommands: []cli.Command{
					{
						Name:        "set",
						Description: "update advanced rack parameters",
						Usage:       "NAME=VALUE [NAME=VALUE]",
						Action:      cmdRackParamsSet,
						Flags:       []cli.Flag{appFlag},
					},
				},
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
						Usage: "vertically scale the instance type, e.g. t2.small or c3.xlarge",
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
	fmt.Printf("Region   %s\n", system.Region)
	fmt.Printf("Count    %d\n", system.Count)
	fmt.Printf("Type     %s\n", system.Type)
}

func cmdRackParams(c *cli.Context) {
	system, err := rackClient(c).GetSystem()

	if err != nil {
		stdcli.Error(err)
		return
	}

	params, err := rackClient(c).ListParameters(system.Name)

	if err != nil {
		stdcli.Error(err)
		return
	}

	keys := []string{}

	for key, _ := range params {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	t := stdcli.NewTable("NAME", "VALUE")

	for _, key := range keys {
		t.AddRow(key, params[key])
	}

	t.Print()
}

func cmdRackParamsSet(c *cli.Context) {
	system, err := rackClient(c).GetSystem()

	if err != nil {
		stdcli.Error(err)
		return
	}

	params := map[string]string{}

	for _, arg := range c.Args() {
		parts := strings.SplitN(arg, "=", 2)

		if len(parts) != 2 {
			stdcli.Error(fmt.Errorf("invalid argument: %s", arg))
			return
		}

		params[parts[0]] = parts[1]
	}

	fmt.Print("Updating parameters... ")

	err = rackClient(c).SetParameters(system.Name, params)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Println("OK")
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

	fmt.Println()
	fmt.Printf("Updating to version: %s\n", version.Version)
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
	system, err := rackClient(c).GetSystem()

	if err != nil {
		stdcli.Error(err)
		return
	}

	pendingVersion := system.Version

	releases, err := rackClient(c).GetSystemReleases()

	if err != nil {
		stdcli.Error(err)
		return
	}

	t := stdcli.NewTable("VERSION", "UPDATED", "STATUS")

	for i, r := range releases {
		status := ""

		if system.Status == "updating" && i == 0 {
			pendingVersion = r.Id
			status = "updating"
		}

		if system.Version == r.Id {
			status = "active"
		}

		t.AddRow(r.Id, humanizeTime(r.Created), status)
	}

	t.Print()

	next, err := version.Next(system.Version)

	if err != nil {
		return
	}

	if next > pendingVersion {
		// if strings.Compare(next, pendingVersion) == 1 {
		fmt.Println()
		fmt.Printf("New version available: %s\n", next)
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
