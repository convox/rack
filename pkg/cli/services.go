package cli

import (
	"fmt"
	"strings"

	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/sdk"
	"github.com/convox/stdcli"
)

func init() {
	register("services", "list services for an app", Services, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagApp, flagRack},
		Validate: stdcli.Args(0),
	})

	register("services restart", "restart a service", ServicesRestart, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagApp, flagRack},
		Validate: stdcli.Args(1),
	})
}

func Services(rack sdk.Interface, c *stdcli.Context) error {
	sys, err := rack.SystemGet()
	if err != nil {
		return err
	}

	var ss structs.Services

	if sys.Version < "20180708231844" {
		ss, err = rack.FormationGet(app(c))
		if err != nil {
			return err
		}
	} else {
		ss, err = rack.ServiceList(app(c))
		if err != nil {
			return err
		}
	}

	t := c.Table("SERVICE", "DOMAIN", "PORTS")

	for _, s := range ss {
		ports := []string{}

		for _, p := range s.Ports {
			port := fmt.Sprintf("%d", p.Balancer)

			if p.Container != 0 {
				port = fmt.Sprintf("%d:%d", p.Balancer, p.Container)
			}

			ports = append(ports, port)
		}

		t.AddRow(s.Name, s.Domain, strings.Join(ports, " "))
	}

	return t.Print()
}

func ServicesRestart(rack sdk.Interface, c *stdcli.Context) error {
	name := c.Arg(0)

	c.Startf("Restarting <service>%s</service>", name)

	if err := rack.ServiceRestart(app(c), name); err != nil {
		return err
	}

	return c.OK()
}
