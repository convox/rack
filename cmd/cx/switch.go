package main

import (
	"github.com/convox/stdcli"
)

func init() {
	CLI.Command("switch", "switch current rack", Switch, stdcli.CommandOptions{
		Validate: stdcli.ArgsMax(1),
	})
}

func Switch(c *stdcli.Context) error {
	if rack := c.Arg(0); rack != "" {
		r, err := matchRack(c, rack)
		if err != nil {
			return err
		}

		if err := c.SettingWrite("rack", r.Name); err != nil {
			return err
		}

		c.Writef("Switched to <rack>%s</rack>\n", r.Name)

		return nil
	}

	rack, err := currentRack(c)
	if err != nil {
		return err
	}

	c.Writef("%s\n", rack)

	return nil
}
