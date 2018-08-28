package main

import "github.com/convox/stdcli"

func init() {
	CLI.Command("registries", "list private registries", Registries, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack},
		Validate: stdcli.Args(0),
	})

	CLI.Command("registries add", "add a private registry", RegistriesAdd, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack},
		Usage:    "<server> <username> <password>",
		Validate: stdcli.Args(3),
	})

	CLI.Command("registries remove", "remove private registry", RegistriesRemove, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack},
		Validate: stdcli.Args(1),
	})
}

func Registries(c *stdcli.Context) error {
	rs, err := provider(c).RegistryList()
	if err != nil {
		return err
	}

	t := c.Table("SERVER", "USERNAME")

	for _, r := range rs {
		t.AddRow(r.Server, r.Username)
	}

	return t.Print()
}

func RegistriesAdd(c *stdcli.Context) error {
	c.Startf("Adding registry")

	if _, err := provider(c).RegistryAdd(c.Arg(0), c.Arg(1), c.Arg(2)); err != nil {
		return err
	}

	return c.OK()
}

func RegistriesRemove(c *stdcli.Context) error {
	s, err := provider(c).SystemGet()
	if err != nil {
		return err
	}

	c.Startf("Removing registry")

	if s.Version <= "20180708231844" {
		if err := provider(c).RegistryRemoveClassic(c.Arg(0)); err != nil {
			return err
		}
	} else {
		if err := provider(c).RegistryRemove(c.Arg(0)); err != nil {
			return err
		}
	}

	return c.OK()
}
