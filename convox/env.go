package main

import (
	"fmt"
	"os"
	"path"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "env",
		Description: "manage an app's environment variables",
		Usage:       "set|change|delete",
		Subcommands: []cli.Command{
			{
				Name:   "get",
				Usage:  "",
				Action: cmdEnvGet,
			},
			{
				Name:   "set",
				Usage:  "VARIABLE=VALUE",
				Action: cmdEnvSet,
			},
			{
				Name:   "unset",
				Usage:  "VARIABLE",
				Action: cmdEnvUnset,
			},
		},
	})
}

func cmdEnvGet(c *cli.Context) {
	variable := c.Args()[0]

	appName := dir()
	path := fmt.Sprintf("apps/%s/environment", appName)

	resp, err := ConvoxGet(path)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Println(resp)
}

func cmdEnvSet(c *cli.Context) {
	data := ""

	for _, value := range c.Args() {
		data += fmt.Sprintf("%s\n", value)
	}

	appName := dir()

	path := fmt.Sprintf("apps/%s/environment", appName)

	resp, err := ConvoxPost(path, data)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Println(resp)
}

func cmdEnvUnset(c *cli.Context) {
	variable := c.Args()[0]

	appName := dir()

	path := fmt.Sprintf("apps/%s/environment/%s", appName, variable)

	resp, err := ConvoxDelete(path)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Println(resp)
}

func dir() string {
	wd, err := os.Getwd()

	if err != nil {
		panic(err)
	}

	return path.Base(wd)
}
