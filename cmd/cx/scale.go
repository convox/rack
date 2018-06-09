package main

import (
	"fmt"

	"github.com/convox/rack/structs"
	"github.com/convox/stdcli"
)

func init() {
	CLI.Command("scale", "scale an rack", Scale, stdcli.CommandOptions{
		Flags: append(stdcli.OptionFlags(structs.ServiceUpdateOptions{}), flagApp, flagRack),
		Validate: func(c *stdcli.Context) error {
			if c.Value("count") != nil || c.Value("cpu") != nil || c.Value("memory") != nil {
				if len(c.Args) < 1 {
					return fmt.Errorf("service name required")
				} else {
					return stdcli.Args(1)(c)
				}
			} else {
				return stdcli.Args(0)(c)
			}
		},
	})
}

func Scale(c *stdcli.Context) error {
	var opts structs.ServiceUpdateOptions

	if err := c.Options(&opts); err != nil {
		return err
	}

	if opts.Count != nil || opts.Cpu != nil || opts.Memory != nil {
		service := c.Arg(0)

		c.Startf("Scaling <service>%s</service>", service)

		if err := provider(c).ServiceUpdate(app(c), service, opts); err != nil {
			return err
		}

		return c.OK()
	}

	ss, err := provider(c).ServiceList(app(c))
	if err != nil {
		return err
	}

	ps, err := provider(c).ProcessList(app(c), structs.ProcessListOptions{})
	if err != nil {
		return err
	}

	running := map[string]int{}

	for _, p := range ps {
		running[p.Name] += 1
	}

	t := c.Table("SERVICE", "DESIRED", "RUNNING", "CPU", "MEMORY")

	for _, s := range ss {
		t.AddRow(s.Name, fmt.Sprintf("%d", s.Count), fmt.Sprintf("%d", running[s.Name]), fmt.Sprintf("%d", s.Cpu), fmt.Sprintf("%d", s.Memory))
	}

	return t.Print()
}
