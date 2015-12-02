package main

import (
	"fmt"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/rack/cmd/convox/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "registries",
		Action:      cmdRegistryList,
		Description: "manage private registries",
		Flags: []cli.Flag{
			appFlag,
		},
		Subcommands: []cli.Command{
			{
				Name:        "add",
				Description: "add a new registry",
				Usage:       "[server]",
				Action:      cmdRegistryAdd,
				Flags: []cli.Flag{
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
			},
		},
	})
}

func cmdRegistryAdd(c *cli.Context) {
	if len(c.Args()) < 1 {
		stdcli.Usage(c, "add")
		return
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
		stdcli.Error(err)
		return
	}

	fmt.Println("Done.")
}

func cmdRegistryList(c *cli.Context) {
	registries, err := rackClient(c).ListRegistries()

	if err != nil {
		stdcli.Error(err)
		return
	}

	t := stdcli.NewTable("SERVER")

	for _, reg := range *registries {
		t.AddRow(reg.ServerAddress)
	}

	t.Print()
}

func cmdRegistryRemove(c *cli.Context) {
	if len(c.Args()) < 1 {
		stdcli.Usage(c, "remove")
		return
	}

	server := c.Args()[0]

	_, err := rackClient(c).RemoveRegistry(server)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Println("Done.")
}
