package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/convox/rack/cmd/convox/stdcli"
	"gopkg.in/urfave/cli.v1"
)

var endpoints = []string{
	"/apps",
	"/apps/<app-name>",
	"/auth",
	"/certificates",
	"/index",
	"/instances",
	"/racks",
	"/registries",
	"/resources",
	"/switch",
	"/system",
}

var apiHelp = fmt.Sprintf(`Valid endpoints:
  %s

For more information, see https://convox.com/api`,
	strings.Join(endpoints, "\n  "))

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "api",
		Description: "make a rest api call to a convox endpoint",
		Usage:       "<command> <endpoint> [options]",
		ArgsUsage:   "<command> <endpoint>",
		Action:      cmdApi,
		Flags:       []cli.Flag{rackFlag},
		Subcommands: []cli.Command{
			{
				Name:        "get",
				Description: "make a GET request to an api endpoint",
				Usage:       "<endpoint> [options]",
				UsageText:   apiHelp,
				ArgsUsage:   "<endpoint>",
				Action:      cmdApiGet,
				Flags:       []cli.Flag{rackFlag},
			},
			{
				Name:        "delete",
				Description: "make a DELETE request to an api endpoint",
				Usage:       "delete an api endpoint",
				UsageText:   apiHelp,
				ArgsUsage:   "<endpoint>",
				Action:      cmdApiDelete,
				Flags:       []cli.Flag{rackFlag},
			},
		},
	})
}

func cmdApi(c *cli.Context) error {
	stdcli.NeedHelp(c)
	stdcli.NeedArg(c, 0)

	// If we're here, it means the user has run 'convox api' without any subcommand or help flag
	stdcli.Errorf("Missing expected subcommand")
	cli.ShowCommandHelp(c, c.Command.Name)

	// Also print the list of endpoints for good measure
	fmt.Printf("\n%s\n", apiHelp)
	return nil
}

func cmdApiGet(c *cli.Context) error {
	stdcli.NeedHelp(c)
	stdcli.NeedArg(c, 1)

	path := c.Args()[0]
	path = strings.TrimRight(path, "/")

	var object interface{}

	err := rackClient(c).Get(path, &object)
	if err != nil {
		return stdcli.Error(err)
	}

	data, err := json.MarshalIndent(object, "", "  ")
	if err != nil {
		return stdcli.Error(err)
	}

	fmt.Println(string(data))
	return nil
}

func cmdApiDelete(c *cli.Context) error {
	stdcli.NeedHelp(c)
	stdcli.NeedArg(c, 1)

	path := c.Args()[0]
	path = strings.TrimRight(path, "/")

	var object interface{}

	err := rackClient(c).Delete(path, &object)
	if err != nil {
		return stdcli.Error(err)
	}

	data, err := json.MarshalIndent(object, "", "  ")
	if err != nil {
		return stdcli.Error(err)
	}

	fmt.Println(string(data))
	return nil
}
