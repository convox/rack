package main

import (
	"encoding/json"
	"fmt"

	"github.com/convox/rack/cmd/convox/stdcli"
	"gopkg.in/urfave/cli.v1"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "api",
		Description: "api endpoint",
		Usage:       "",
		Action:      cmdApi,
		Flags:       []cli.Flag{rackFlag},
		Subcommands: []cli.Command{
			{
				Name:        "get",
				Description: "get an api endpoint",
				Usage:       "<endpoint>",
				Action:      cmdApiGet,
				Flags:       []cli.Flag{rackFlag},
			},
			{
				Name:        "delete",
				Description: "delete an api endpoint",
				Usage:       "<endpoint>",
				Action:      cmdApiDelete,
				Flags:       []cli.Flag{rackFlag},
			},
		},
	})
}

func cmdApi(c *cli.Context) error {
	if len(c.Args()) > 0 {
		return cli.NewExitError("ERROR: `convox api` does not take arguments. Perhaps you meant `convox api get`?", 1)
	}

	stdcli.Usage(c, "")
	return nil
}

func cmdApiGet(c *cli.Context) error {
	if len(c.Args()) < 1 {
		stdcli.Usage(c, "get")
		return nil
	}

	path := c.Args()[0]

	var object interface{}

	err := rackClient(c).Get(path, &object)
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	data, err := json.MarshalIndent(object, "", "  ")
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	fmt.Println(string(data))
	return nil
}

func cmdApiDelete(c *cli.Context) error {
	if len(c.Args()) < 1 {
		stdcli.Usage(c, "delete")
		return nil
	}

	path := c.Args()[0]

	var object interface{}

	err := rackClient(c).Delete(path, &object)
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	data, err := json.MarshalIndent(object, "", "  ")
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	fmt.Println(string(data))
	return nil
}
