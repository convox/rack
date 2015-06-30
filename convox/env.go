package main

import (
	"encoding/json"
	"fmt"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
)

var appFlag = cli.StringFlag{
	Name:  "app",
	Usage: "app name. Inferred from current directory if not specified.",
}

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "env",
		Description: "manage an app's environment variables",
		Usage:       "get|set|unset",
		Action:      cmdEnvGetAll,
		Flags:       []cli.Flag{appFlag},
		Subcommands: []cli.Command{
			{
				Name:   "get",
				Usage:  "VARIABLE",
				Action: cmdEnvGet,
				Flags:  []cli.Flag{appFlag},
			},
			{
				Name:   "set",
				Usage:  "VARIABLE=VALUE",
				Action: cmdEnvSet,
				Flags:  []cli.Flag{appFlag},
			},
			{
				Name:   "unset",
				Usage:  "VARIABLE",
				Action: cmdEnvUnset,
				Flags:  []cli.Flag{appFlag},
			},
		},
	})
}

func cmdEnvGetAll(c *cli.Context) {
	app := c.String("app")

	if app == "" {
		app = DirAppName()
	}

	resp, err := fetchEnv(app)

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
	app := c.String("app")

	if app == "" {
		app = DirAppName()
	}

	variable := c.Args()[0]

	resp, err := fetchEnv(app)

	if err != nil {
		stdcli.Error(err)
		return
	}

	var env map[string]string
	json.Unmarshal(resp, &env)

	fmt.Println(env[variable])
}

func cmdEnvSet(c *cli.Context) {
	app := c.String("app")

	if app == "" {
		app = DirAppName()
	}

	resp, err := fetchEnv(app)

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

	path := fmt.Sprintf("/apps/%s/environment", app)

	resp, err = ConvoxPost(path, data)

	if err != nil {
		stdcli.Error(err)
		return
	}
}

func cmdEnvUnset(c *cli.Context) {
	app := c.String("app")

	if app == "" {
		app = DirAppName()
	}
	variable := c.Args()[0]

	path := fmt.Sprintf("/apps/%s/environment/%s", app, variable)

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
