package main

import (
	"fmt"
	"os/user"

	"github.com/convox/rack/router"
	"github.com/convox/stdcli"
)

func init() {
	CLI.Command("router", "start local router", Router, stdcli.CommandOptions{
		Flags: []stdcli.Flag{
			stdcli.StringFlag("interface", "i", "interface name"),
			stdcli.StringFlag("subnet", "s", "subnet cidr"),
		},
		Invisible: true,
		Validate:  stdcli.Args(0),
	})
}

func Router(c *stdcli.Context) error {
	u, err := user.Current()
	if err != nil {
		return err
	}

	if u.Uid != "0" {
		return fmt.Errorf("must run as root")
	}

	iface := coalesce(c.String("interface"), "vlan2")
	subnet := coalesce(c.String("subnet"), "10.42.0.0/16")

	r, err := router.New(iface, subnet, version)
	if err != nil {
		return err
	}

	if err := r.Serve(); err != nil {
		return err
	}

	return nil
}
