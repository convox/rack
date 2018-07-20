package main

import (
	"encoding/json"

	"github.com/convox/stdcli"
)

func init() {
	CLI.Command("switch", "switch current rack", Switch, stdcli.CommandOptions{
		Validate: stdcli.ArgsMax(1),
	})
}

func Switch(c *stdcli.Context) error {
	host, err := currentHost(c)
	if err != nil {
		return err
	}

	if rack := c.Arg(0); rack != "" {
		r, err := matchRack(c, rack)
		if err != nil {
			return err
		}

		rs := hostRacks(c)

		rs[host] = r.Name

		data, err := json.MarshalIndent(rs, "", "  ")
		if err != nil {
			return err
		}

		if err := c.SettingWrite("racks", string(data)); err != nil {
			return err
		}

		c.Writef("Switched to <rack>%s</rack>\n", r.Name)

		return nil
	}

	rack := currentRack(c, host)

	c.Writef("%s\n", rack)

	return nil
}
