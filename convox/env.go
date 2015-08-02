package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"sort"

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
	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	resp, err := fetchEnv(app)

	if err != nil {
		stdcli.Error(err)
		return
	}

	var env map[string]string
	json.Unmarshal(resp, &env)

	keys := []string{}

	for key, _ := range env {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		fmt.Printf("%s=%s\n", key, env[key])
	}
}

func cmdEnvGet(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	
	if len(c.Args()) == 0 {
		stdcli.Error(errors.New("No variable specified))
		return
	}
	
	if len(c.Args()) > 1 {
		stdcli.Error(errors.New("Only 1 variable can be retrieved at a time))
		return
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
	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
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

	stat, _ := os.Stdin.Stat()

	if (stat.Mode() & os.ModeCharDevice) == 0 {
		in, err := ioutil.ReadAll(os.Stdin)

		if err != nil {
			stdcli.Error(err)
			return
		}

		data += string(in)
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
	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}
	
	if len(c.Args()) == 0 {
		stdcli.Error(errors.New("No variable specified))
		return
	}
	
	if len(c.Args()) > 1 {
		stdcli.Error(errors.New("Only 1 variable can be unset at a time))
		return
	}

	variable := c.Args()[0]

	path := fmt.Sprintf("/apps/%s/environment/%s", app, variable)

	_, err = ConvoxDelete(path)

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
