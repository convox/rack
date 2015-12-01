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
		Description: "manage image registries",
		Flags: []cli.Flag{
			appFlag,
		},
		Subcommands: []cli.Command{
			{
				Name:        "add",
				Description: "add a new registry",
				Usage:       "[hostname]",
				Action:      cmdRegistryAdd,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "username, u",
						Usage: "Username for authentication. If not specified, prompt for username and password.",
					},
					cli.StringFlag{
						EnvVar: "PASSWORD",
						Name:   "password, p",
						Usage:  "Password for authentication. If not specified, prompt for username and password.",
					},
				},
			},
			{
				Name:        "remove",
				Description: "remove a registry",
				Usage:       "[hostname]",
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

	host := c.Args()[0]
	username := c.String("username")
	password := c.String("password")

	if username == "" {
		username = promptForUsername()
	}

	if password == "" {
		password = promptForPassword()
	}

	_, err := rackClient(c).AddRegistry(host, username, password)

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

	t := stdcli.NewTable("HOST", "USER")

	for _, reg := range *registries {
		t.AddRow(reg.ServerAddress, reg.Username)
	}

	t.Print()
}

func cmdRegistryRemove(c *cli.Context) {
	if len(c.Args()) < 1 {
		stdcli.Usage(c, "remove")
		return
	}

	host := c.Args()[0]

	_, err := rackClient(c).RemoveRegistry(host)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Println("Done.")
}
