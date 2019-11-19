package cli

import (
	"sort"

	"github.com/convox/rack/sdk"
	"github.com/convox/stdcli"
)

func init() {
	register("restart", "restart an app", Restart, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagApp, flagRack},
		Validate: stdcli.Args(0),
	})
}

func Restart(rack sdk.Interface, c *stdcli.Context) error {
	ss, err := rack.ServiceList(app(c))
	if err != nil {
		return err
	}

	sort.Slice(ss, func(i, j int) bool { return ss[i].Name < ss[j].Name })

	for _, s := range ss {
		c.Startf("Restarting <service>%s</service>", s.Name)

		if err := rack.ServiceRestart(app(c), s.Name); err != nil {
			return err
		}

		c.OK()
	}

	return nil
}
