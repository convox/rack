package cli

import (
	"fmt"
	"sort"

	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/sdk"
	"github.com/convox/stdcli"
)

func init() {
	register("scale", "scale a service", Scale, stdcli.CommandOptions{
		Flags: append(stdcli.OptionFlags(structs.ServiceUpdateOptions{}), flagApp, flagRack, flagWait),
		Usage: "<service>",
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

func Scale(rack sdk.Interface, c *stdcli.Context) error {
	s, err := rack.SystemGet()
	if err != nil {
		return err
	}

	var opts structs.ServiceUpdateOptions

	if err := c.Options(&opts); err != nil {
		return err
	}

	if opts.Count != nil || opts.Cpu != nil || opts.Memory != nil {
		service := c.Arg(0)

		c.Startf("Scaling <service>%s</service>", service)

		if s.Version <= "20180708231844" {
			if err := rack.FormationUpdate(app(c), service, opts); err != nil {
				return err
			}
		} else {
			if err := rack.ServiceUpdate(app(c), service, opts); err != nil {
				return err
			}
		}

		if c.Bool("wait") {
			if err := waitForAppWithLogs(rack, c, app(c)); err != nil {
				return err
			}
		}

		return c.OK()
	}

	var ss structs.Services
	running := map[string]int{}

	if s.Version < "20180708231844" {
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

	sort.Slice(ss, func(i, j int) bool { return ss[i].Name < ss[j].Name })

	ps, err := rack.ProcessList(app(c), structs.ProcessListOptions{})
	if err != nil {
		return err
	}

	for _, p := range ps {
		running[p.Name] += 1
	}

	t := c.Table("SERVICE", "DESIRED", "RUNNING", "CPU", "MEMORY")

	for _, s := range ss {
		t.AddRow(s.Name, fmt.Sprintf("%d", s.Count), fmt.Sprintf("%d", running[s.Name]), fmt.Sprintf("%d", s.Cpu), fmt.Sprintf("%d", s.Memory))
	}

	return t.Print()
}
