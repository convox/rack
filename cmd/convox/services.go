package main

import (
	"fmt"
	"strings"

	"github.com/convox/stdcli"
)

func init() {
	CLI.Command("services", "list services for an app", Services, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagApp, flagRack},
		Validate: stdcli.Args(0),
	})
}

func Services(c *stdcli.Context) error {
	ss, err := provider(c).ServiceList(app(c))
	if err != nil {
		return err
	}

	t := c.Table("SERVICE", "DOMAIN", "PORTS")

	for _, s := range ss {
		ports := []string{}

		for _, p := range s.Ports {
			ports = append(ports, fmt.Sprintf("%d:%d", p.Balancer, p.Container))
		}

		t.AddRow(s.Name, s.Domain, strings.Join(ports, " "))
	}

	return t.Print()
}
