package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"

	"github.com/convox/rack/cmd/convox/stdcli"
	"gopkg.in/urfave/cli.v1"
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

func cmdEnvList(c *cli.Context) error {
	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.ExitError(err)
	}

	if len(c.Args()) > 0 {
		return stdcli.ExitError(fmt.Errorf("`convox env` does not take arguments. Perhaps you meant `convox env set`?"))
	}

	if c.Bool("help") {
		stdcli.Usage(c, "")
		return nil
	}

	env, err := rackClient(c).GetEnvironment(app)
	if err != nil {
		return stdcli.ExitError(err)
	}

	keys := []string{}

	for key, _ := range env {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		fmt.Printf("%s=%s\n", key, env[key])
	}

	return nil
}

func cmdEnvGet(c *cli.Context) error {
	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.ExitError(err)
	}

	if len(c.Args()) == 0 {
		return stdcli.ExitError(errors.New("No variable specified"))
	}

	if len(c.Args()) > 1 {
		return stdcli.ExitError(errors.New("Only 1 variable can be retrieved at a time"))
	}

	variable := c.Args()[0]

	env, err := rackClient(c).GetEnvironment(app)
	if err != nil {
		return stdcli.ExitError(err)
	}

	fmt.Println(env[variable])
	return nil
}

func cmdEnvSet(c *cli.Context) error {
	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.ExitError(err)
	}

	env, err := rackClient(c).GetEnvironment(app)
	if err != nil {
		return stdcli.ExitError(err)
	}

	data := ""

	for key, value := range env {
		data += fmt.Sprintf("%s=%s\n", key, value)
	}

	stat, err := os.Stdin.Stat()
	if err != nil {
		return stdcli.ExitError(err)
	}

	if (stat.Mode() & os.ModeCharDevice) == 0 {
		in, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			return stdcli.ExitError(err)
		}

		data += string(in)
	}

	for _, value := range c.Args() {
		data += fmt.Sprintf("%s\n", value)
	}

	fmt.Print("Updating environment... ")

	_, releaseId, err := rackClient(c).SetEnvironment(app, strings.NewReader(data))
	if err != nil {
		return stdcli.ExitError(err)
	}

	fmt.Println("OK")

	if releaseId != "" {
		if c.Bool("promote") {
			fmt.Printf("Promoting %s... ", releaseId)

			_, err = rackClient(c).PromoteRelease(app, releaseId)
			if err != nil {
				return stdcli.ExitError(err)
			}

			fmt.Println("OK")
		} else {
			fmt.Printf("To deploy these changes run `convox releases promote %s`\n", releaseId)
		}
	}

	return nil
}

func cmdEnvUnset(c *cli.Context) error {
	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.ExitError(err)
	}

	if len(c.Args()) == 0 {
		return stdcli.ExitError(errors.New("No variable specified"))
	}

	if len(c.Args()) > 1 {
		return stdcli.ExitError(errors.New("Only 1 variable can be unset at a time"))
	}

	key := c.Args()[0]

	fmt.Print("Updating environment... ")

	_, releaseId, err := rackClient(c).DeleteEnvironment(app, key)
	if err != nil {
		return stdcli.ExitError(err)
	}

	fmt.Println("OK")

	if releaseId != "" {
		if c.Bool("promote") {
			fmt.Printf("Promoting %s... ", releaseId)

			_, err = rackClient(c).PromoteRelease(app, releaseId)
			if err != nil {
				return stdcli.ExitError(err)
			}

			fmt.Println("OK")
		} else {
			fmt.Printf("To deploy these changes run `convox releases promote %s`\n", releaseId)
		}
	}

	return nil
}
