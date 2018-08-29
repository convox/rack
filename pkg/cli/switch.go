package cli

import (
	"encoding/json"

	"github.com/convox/rack/sdk"
	"github.com/convox/stdcli"
)

func init() {
	register("switch", "switch current rack", Switch, stdcli.CommandOptions{
		Validate: stdcli.ArgsMax(1),
	})
}

func Switch(rack sdk.Interface, c *stdcli.Context) error {
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

	c.Writef("%s\n", currentRack(c, host))

	return nil
}
