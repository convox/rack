package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/convox/rack/cmd/convox/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "env",
		Description: "manage an app's environment variables",
		Usage:       "",
		Action:      cmdEnvList,
		Flags:       []cli.Flag{appFlag},
		Subcommands: []cli.Command{
			{
				Name:        "get",
				Description: "get all environment variables",
				Usage:       "VARIABLE",
				Action:      cmdEnvGet,
				Flags:       []cli.Flag{appFlag},
			},
			{
				Name:        "set",
				Description: "set an environment variable",
				Usage:       "VARIABLE=VALUE",
				Action:      cmdEnvSet,
				Flags: []cli.Flag{
					appFlag,
					cli.BoolFlag{
						Name:  "promote",
						Usage: "promote the release after env change",
					},
				},
			},
			{
				Name:        "unset",
				Description: "delete an environment varible",
				Usage:       "VARIABLE",
				Action:      cmdEnvUnset,
				Flags: []cli.Flag{
					appFlag,
					cli.BoolFlag{
						Name:  "promote",
						Usage: "promote the release after env change",
					},
				},
			},
		},
	})
}

func cmdEnvList(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		stdcli.Error(err)
		return
	}

	if len(c.Args()) > 0 {
		stdcli.Error(fmt.Errorf("`convox env` does not take arguments. Perhaps you meant `convox env set`?"))
		return
	}

	if c.Bool("help") {
		stdcli.Usage(c, "")
		return
	}

	env, err := rackClient(c).GetEnvironment(app)

	if err != nil {
		stdcli.Error(err)
		return
	}

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
		stdcli.Error(errors.New("No variable specified"))
		return
	}

	if len(c.Args()) > 1 {
		stdcli.Error(errors.New("Only 1 variable can be retrieved at a time"))
		return
	}

	variable := c.Args()[0]

	env, err := rackClient(c).GetEnvironment(app)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Println(env[variable])
}

func cmdEnvSet(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		stdcli.Error(err)
		return
	}

	env, err := rackClient(c).GetEnvironment(app)

	if err != nil {
		stdcli.Error(err)
		return
	}

	data := ""

	for key, value := range env {
		data += fmt.Sprintf("%s=%s\n", key, value)
	}

	stat, err := os.Stdin.Stat()

	if err != nil {
		stdcli.Error(err)
		return
	}

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

	fmt.Print("Updating environment... ")

	_, releaseId, err := rackClient(c).SetEnvironment(app, strings.NewReader(data))

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Println("OK")

	if releaseId != "" {
		if c.Bool("promote") {
			fmt.Printf("Promoting %s... ", releaseId)

			_, err = rackClient(c).PromoteRelease(app, releaseId)

			if err != nil {
				stdcli.Error(err)
				return
			}

			fmt.Println("OK")
		} else {
			fmt.Printf("To deploy these changes run `convox releases promote %s`\n", releaseId)
		}
	}
}

func cmdEnvUnset(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		stdcli.Error(err)
		return
	}

	if len(c.Args()) == 0 {
		stdcli.Error(errors.New("No variable specified"))
		return
	}

	if len(c.Args()) > 1 {
		stdcli.Error(errors.New("Only 1 variable can be unset at a time"))
		return
	}

	key := c.Args()[0]

	fmt.Print("Updating environment... ")

	_, releaseId, err := rackClient(c).DeleteEnvironment(app, key)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Println("OK")

	if releaseId != "" {
		if c.Bool("promote") {
			fmt.Printf("Promoting %s... ", releaseId)

			_, err = rackClient(c).PromoteRelease(app, releaseId)

			if err != nil {
				stdcli.Error(err)
				return
			}

			fmt.Println("OK")
		} else {
			fmt.Printf("To deploy these changes run `convox releases promote %s`\n", releaseId)
		}
	}
}
