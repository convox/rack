package cli

import (
	"github.com/convox/rack/sdk"
	"github.com/convox/stdcli"
)

func init() {
	register("registries", "list private registries", Registries, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack},
		Validate: stdcli.Args(0),
	})

	register("registries add", "add a private registry", RegistriesAdd, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack},
		Usage:    "<server> <username> <password>",
		Validate: stdcli.Args(3),
	})

	register("registries remove", "remove private registry", RegistriesRemove, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack},
		Validate: stdcli.Args(1),
	})
}

func Registries(rack sdk.Interface, c *stdcli.Context) error {
	rs, err := rack.RegistryList()
	if err != nil {
		return err
	}

	t := c.Table("SERVER", "USERNAME")

	for _, r := range rs {
		t.AddRow(r.Server, r.Username)
	}

	return t.Print()
}

func RegistriesAdd(rack sdk.Interface, c *stdcli.Context) error {
	c.Startf("Adding registry")

	if _, err := rack.RegistryAdd(c.Arg(0), c.Arg(1), c.Arg(2)); err != nil {
		return err
	}

	return c.OK()
}

func RegistriesRemove(rack sdk.Interface, c *stdcli.Context) error {
	s, err := rack.SystemGet()
	if err != nil {
		return err
	}

	c.Startf("Removing registry")

	if s.Version <= "20180708231844" {
		if err := rack.RegistryRemoveClassic(c.Arg(0)); err != nil {
			return err
		}
	} else {
		if err := rack.RegistryRemove(c.Arg(0)); err != nil {
			return err
		}
	}

	return c.OK()
}
