package main

import (
	"fmt"

	"github.com/convox/rack/cmd/convox/stdcli"
	"gopkg.in/urfave/cli.v1"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "registries",
		Action:      cmdRegistryList,
		Description: "manage private registries",
		Flags:       []cli.Flag{rackFlag},
		Subcommands: []cli.Command{
			{
				Name:        "add",
				Description: "add a new registry",
				Usage:       "[server]",
				Action:      cmdRegistryAdd,
				Flags: []cli.Flag{
					rackFlag,
					cli.StringFlag{
						Name:  "email, e",
						Usage: "Email for registry auth.",
					},
					cli.StringFlag{
						Name:  "username, u",
						Usage: "Username for auth. If not specified, prompt for username.",
					},
					cli.StringFlag{
						EnvVar: "PASSWORD",
						Name:   "password, p",
						Usage:  "Password for auth. If not specified, prompt for password.",
					},
				},
			},
			{
				Name:        "remove",
				Description: "remove a registry",
				Usage:       "[server]",
				Action:      cmdRegistryRemove,
				Flags:       []cli.Flag{rackFlag},
			},
		},
	})
}

func cmdRegistryAdd(c *cli.Context) error {
	if len(c.Args()) < 1 {
		stdcli.Usage(c, "add")
		return nil
	}

	server := c.Args()[0]
	username := c.String("username")
	password := c.String("password")
	email := c.String("email")

	if username == "" {
		username = promptForUsername()
	}

	if password == "" {
		password = promptForPassword()
	}

	_, err := rackClient(c).AddRegistry(server, username, password, email)
	if err != nil {
		return stdcli.ExitError(err)
	}

	fmt.Println("Done.")
	return nil
}

func cmdRegistryList(c *cli.Context) error {
	if len(c.Args()) > 0 {
		return stdcli.ExitError(fmt.Errorf("`convox registries` does not take arguments. Perhaps you meant `convox registries add`?"))
	}

	if c.Bool("help") {
		stdcli.Usage(c, "")
		return nil
	}

	registries, err := rackClient(c).ListRegistries()
	if err != nil {
		return stdcli.ExitError(err)
	}

	t := stdcli.NewTable("SERVER")

	for _, reg := range *registries {
		t.AddRow(reg.ServerAddress)
	}

	t.Print()
	return nil
}

func cmdRegistryRemove(c *cli.Context) error {
	if len(c.Args()) < 1 {
		stdcli.Usage(c, "remove")
		return nil
	}

	server := c.Args()[0]

	_, err := rackClient(c).RemoveRegistry(server)
	if err != nil {
		return stdcli.ExitError(err)
	}

	fmt.Println("Done.")
	return nil
}
