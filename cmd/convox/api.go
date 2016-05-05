package main

import (
	"encoding/json"
	"fmt"

	"github.com/codegangsta/cli"
	"github.com/convox/rack/cmd/convox/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "api",
		Description: "api endpoint",
		Usage:       "",
		Action:      cmdApi,
		Subcommands: []cli.Command{
			{
				Name:        "get",
				Description: "get an api endpoint",
				Usage:       "<endpoint>",
				Action:      cmdApiGet,
			},
			{
				Name:        "delete",
				Description: "delete an api endpoint",
				Usage:       "<endpoint>",
				Action:      cmdApiDelete,
			},
		},
	})
}

func cmdApi(c *cli.Context) {
	if len(c.Args()) > 0 {
		stdcli.Error(fmt.Errorf("`convox api` does not take arguments. Perhaps you meant `convox api get`?"))
		return
	}

	stdcli.Usage(c, "")
}

func cmdApiGet(c *cli.Context) {
	if len(c.Args()) < 1 {
		stdcli.Usage(c, "get")
		return
	}

	path := c.Args()[0]

	var object interface{}

	err := rackClient(c).Get(path, &object)

	if err != nil {
		stdcli.Error(err)
	}

	data, err := json.MarshalIndent(object, "", "  ")

	if err != nil {
		stdcli.Error(err)
	}

	fmt.Println(string(data))
}

func cmdApiDelete(c *cli.Context) {
	if len(c.Args()) < 1 {
		stdcli.Usage(c, "get")
		return
	}

	path := c.Args()[0]

	var object interface{}

	err := rackClient(c).Delete(path, &object)

	if err != nil {
		stdcli.Error(err)
	}

	data, err := json.MarshalIndent(object, "", "  ")

	if err != nil {
		stdcli.Error(err)
	}

	fmt.Println(string(data))
}
