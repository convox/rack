package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/convox/rack/sdk"
	"github.com/convox/stdcli"
)

func init() {
	registerWithoutProvider("switch", "switch current rack", Switch, stdcli.CommandOptions{
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

		var also []string

		if h := selfManagedRackHost(c)[r.Name]; h != "" {
			if os.Getenv("CONVOX_HOST") != "" {
				return fmt.Errorf("unset CONVOX_HOST to switch to %s", r.Name)
			}

			rrs, err := remoteRacks(c)
			if err != nil {
				return err
			}

			if len(rrs) == 0 {
				if err := c.SettingWrite("host", h); err != nil {
					return err
				}

				host = h
			}

			for _, rr := range rrs {
				if strings.Contains(rr.Name, r.Name) {
					also = append(also, rr.Name)
				}
			}
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

		if len(also) > 0 {
			_, _ = fmt.Fprintf(c.Writer().Stderr, "NOTICE: Console racks also match %s: %s. To select one, use its full name from `convox racks`, for example `convox switch %s`\n", r.Name, strings.Join(also, ", "), also[0])
		}

		return nil
	}

	c.Writef("%s\n", currentRack(c, host))

	return nil
}
