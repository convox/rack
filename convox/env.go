package main

import (
	"encoding/json"
	"fmt"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "env",
		Description: "manage an app's environment variables",
		Usage:       "get|set|unset",
		Action:      cmdEnvGetAll,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "name",
				Usage: "app name. Inferred from current directory if not specified.",
			},
		},
		Subcommands: []cli.Command{
			{
				Name:   "get",
				Usage:  "VARIABLE",
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

func cmdEnvGetAll(c *cli.Context) {
	name := c.String("name")

	if name == "" {
		name = DirAppName()
	}

	resp, err := fetchEnv(name)

	if err != nil {
		stdcli.Error(err)
		return
	}

	var env map[string]string
	json.Unmarshal(resp, &env)

	output := ""

	for key, value := range env {
		output += fmt.Sprintf("%s=%s\n", key, value)
	}

	fmt.Print(output)
}

func cmdEnvGet(c *cli.Context) {
	name := c.String("name")

	if name == "" {
		name = DirAppName()
	}

	variable := c.Args()[0]

	resp, err := fetchEnv(name)

	if err != nil {
		stdcli.Error(err)
		return
	}

	var env map[string]string
	json.Unmarshal(resp, &env)

	fmt.Println(env[variable])
}

func cmdEnvSet(c *cli.Context) {
	name := c.String("name")

	if name == "" {
		name = DirAppName()
	}

	resp, err := fetchEnv(name)

	if err != nil {
		stdcli.Error(err)
		return
	}

	var old map[string]string
	json.Unmarshal(resp, &old)

	data := ""

	for key, value := range old {
		data += fmt.Sprintf("%s=%s\n", key, value)
	}

	for _, value := range c.Args() {
		data += fmt.Sprintf("%s\n", value)
	}

	path := fmt.Sprintf("/apps/%s/environment", name)

	resp, err = ConvoxPost(path, data)

	if err != nil {
		stdcli.Error(err)
		return
	}
}

func cmdEnvUnset(c *cli.Context) {
	name := c.String("name")

	if name == "" {
		name = DirAppName()
	}
	variable := c.Args()[0]

	path := fmt.Sprintf("/apps/%s/environment/%s", name, variable)

	_, err := ConvoxDelete(path)

	if err != nil {
		stdcli.Error(err)
		return
	}
}

func fetchEnv(app string) ([]byte, error) {
	path := fmt.Sprintf("/apps/%s/environment", app)

	resp, err := ConvoxGet(path)

	if err != nil {
		return nil, err
	}

	return resp, nil
}
