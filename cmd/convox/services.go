package main

import (
	"fmt"
	"strings"

	"github.com/convox/rack/pkg/structs"
	"github.com/convox/stdcli"
)

func init() {
	CLI.Command("services", "list services for an app", Services, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagApp, flagRack},
		Validate: stdcli.Args(0),
	})
}

func Services(c *stdcli.Context) error {
	sys, err := provider(c).SystemGet()
	if err != nil {
		return err
	}

	var ss structs.Services

	if sys.Version < "20180708231844" {
		ss, err = provider(c).FormationGet(app(c))
		if err != nil {
			return err
		}
	} else {
		ss, err = provider(c).ServiceList(app(c))
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
