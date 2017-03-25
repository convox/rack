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
		UsageText:   "(add|remove)",
		Usage:       "(add|remove) <registry> [--username USERNAME] [--password PASSWORD]",
		ArgsUsage:   "<subcommand>",
		Flags:       []cli.Flag{rackFlag},
		Subcommands: []cli.Command{
			{
				Name:        "add",
				Description: "add a new registry",
				Usage:       "<server> [--username USERNAME] [--password PASSWORD]",
				ArgsUsage:   "<server>",
				UsageText:   "<server> [--username USERNAME] [--password PASSWORD]",
				Action:      cmdRegistryAdd,
				Flags: []cli.Flag{
					rackFlag,
					cli.StringFlag{
						Name:  "email, e",
						Usage: "email for registry auth",
					},
					cli.StringFlag{
						Name:  "username, u",
						Usage: "username for auth. If not specified, prompt for username.",
					},
					cli.StringFlag{
						EnvVar: "PASSWORD",
						Name:   "password, p",
						Usage:  "password for auth. If not specified, prompt for password.",
					},
				},
			},
			{
				Name:        "remove",
				Description: "remove a registry",
				Usage:       "<server>",
				ArgsUsage:   "<server>",
				UsageText:   "<server> (see `convox registries`)",
				Action:      cmdRegistryRemove,
				Flags:       []cli.Flag{rackFlag},
			},
		},
	})
}

func cmdRegistryAdd(c *cli.Context) error {
	stdcli.NeedHelp(c)
	stdcli.NeedArg(c, 1)

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
		return stdcli.Error(err)
	}

	fmt.Println("Done.")
	return nil
}

func cmdRegistryList(c *cli.Context) error {
	stdcli.NeedHelp(c)
	stdcli.NeedArg(c, 0)

	registries, err := rackClient(c).ListRegistries()
	if err != nil {
		return stdcli.Error(err)
	}

	t := stdcli.NewTable("SERVER", "USERNAME")

	for _, reg := range *registries {
		t.AddRow(reg.Server, reg.Username)
	}

	t.Print()
	return nil
}

func cmdRegistryRemove(c *cli.Context) error {
	stdcli.NeedHelp(c)
	stdcli.NeedArg(c, 1)

	server := c.Args()[0]

	err := rackClient(c).RemoveRegistry(server)
	if err != nil {
		return stdcli.Error(err)
	}

	fmt.Println("Done.")
	return nil
}
