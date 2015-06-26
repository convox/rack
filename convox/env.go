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
	appName := dir()

	var env map[string]string
	json.Unmarshal(fetchEnv(appName), &env)

	output := ""

	for key, value := range env {
		output += fmt.Sprintf("%s=%s\n", key, value)
	}

	fmt.Print(output)
}

func cmdEnvGet(c *cli.Context) {
	appName := dir()
	variable := c.Args()[0]

	var env map[string]string
	json.Unmarshal(fetchEnv(appName), &env)

	fmt.Println(env[variable])
}

func cmdEnvSet(c *cli.Context) {
	appName := dir()

	var old map[string]string
	json.Unmarshal(fetchEnv(appName), &old)

	data := ""

	for key, value := range old {
		data += fmt.Sprintf("%s=%s\n", key, value)
	}

	for _, value := range c.Args() {
		data += fmt.Sprintf("%s\n", value)
	}

	path := fmt.Sprintf("/apps/%s/environment", appName)

	resp, err := ConvoxPost(path, data)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Println(string(resp[:]))
}

func cmdEnvUnset(c *cli.Context) {
	variable := c.Args()[0]

	appName := DirAppName()

	path := fmt.Sprintf("/apps/%s/environment/%s", appName, variable)

	resp, err := ConvoxDelete(path)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Println(string(resp[:]))
}

func dir() string {
	wd, err := os.Getwd()

	if err != nil {
		panic(err)
	}

	return path.Base(wd)
}

func fetchEnv(app string) []byte {
	appName := dir()
	path := fmt.Sprintf("/apps/%s/environment", appName)

	resp, err := ConvoxGet(path)

	if err != nil {
		panic(err)
	}

	return resp
}
