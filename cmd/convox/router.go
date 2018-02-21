package main

import (
	"fmt"
	"os/user"

	"github.com/convox/rack/cmd/convox/stdcli"
	"github.com/convox/rack/router"
	"gopkg.in/urfave/cli.v1"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "router",
		Description: "start a local router",
		Action:      runRouter,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "domain, d",
				Usage: "domain name",
				Value: "convox",
			},
			cli.StringFlag{
				Name:  "interface, i",
				Usage: "interface name",
				Value: "vlan2",
			},
			cli.StringFlag{
				Name:  "subnet, s",
				Usage: "subnet",
				Value: "10.42.0.0/16",
			},
		},
	})
}

func runRouter(c *cli.Context) error {
	u, err := user.Current()
	if err != nil {
		return err
	}

	if u.Uid != "0" {
		return fmt.Errorf("must run as root")
	}

	r, err := router.New(Version, c.String("domain"), c.String("interface"), c.String("subnet"))
	if err != nil {
		return err
	}

	if err := r.Serve(); err != nil {
		return err
	}

	return nil
}
