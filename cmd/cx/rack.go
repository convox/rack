package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/convox/rack/helpers"
	"github.com/convox/rack/options"
	"github.com/convox/rack/structs"
	"github.com/convox/stdcli"

	cv "github.com/convox/version"
)

func init() {
	CLI.Command("rack", "get information about the rack", Rack, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack},
		Validate: stdcli.Args(0),
	})

	CLI.Command("rack logs", "get logs for the rack", RackLogs, stdcli.CommandOptions{
		Flags:    append(stdcli.OptionFlags(structs.LogsOptions{}), flagRack),
		Validate: stdcli.Args(0),
	})

	CLI.Command("rack params", "display rack parameters", RackParams, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack},
		Validate: stdcli.Args(0),
	})

	CLI.Command("rack params set", "set rack parameters", RackParamsSet, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack, flagWait},
		Usage:    "<Key=Value> [Key=Value]...",
		Validate: stdcli.ArgsMin(1),
	})

	CLI.Command("rack ps", "list rack processes", RackPs, stdcli.CommandOptions{
		Flags:    append(stdcli.OptionFlags(structs.SystemProcessesOptions{}), flagRack),
		Validate: stdcli.Args(0),
	})

	CLI.Command("rack releases", "list rack version history", RackReleases, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack},
		Validate: stdcli.Args(0),
	})

	CLI.Command("rack scale", "scale the rack", RackScale, stdcli.CommandOptions{
		Flags: []stdcli.Flag{
			flagRack,
			stdcli.IntFlag("count", "c", "instance count"),
			stdcli.StringFlag("type", "t", "instance type"),
		},
		Validate: stdcli.Args(0),
	})

	CLI.Command("rack start", "start local rack", RackStart, stdcli.CommandOptions{
		Flags: []stdcli.Flag{
			stdcli.StringFlag("name", "n", "rack name"),
			stdcli.StringFlag("router", "r", "router host"),
		},
		Validate: stdcli.Args(0),
	})

	CLI.Command("rack update", "update the rack", RackUpdate, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack, flagWait},
		Validate: stdcli.ArgsMax(1),
	})
}

func Rack(c *stdcli.Context) error {
	s, err := provider(c).SystemGet()
	if err != nil {
		return err
	}

	i := c.Info()

	i.Add("Name", s.Name)
	i.Add("Status", s.Status)
	i.Add("Version", s.Version)
	i.Add("Region", s.Region)
	i.Add("Router", s.Domain)

	return i.Print()
}

func RackLogs(c *stdcli.Context) error {
	var opts structs.LogsOptions

	if err := c.Options(&opts); err != nil {
		return err
	}

	r, err := provider(c).SystemLogs(opts)
	if err != nil {
		return err
	}

	io.Copy(c, r)

	return nil
}

func RackParams(c *stdcli.Context) error {
	s, err := provider(c).SystemGet()
	if err != nil {
		return err
	}

	keys := []string{}

	for k := range s.Parameters {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	i := c.Info()

	for _, k := range keys {
		i.Add(k, s.Parameters[k])
	}

	return i.Print()
}

func RackParamsSet(c *stdcli.Context) error {
	opts := structs.SystemUpdateOptions{
		Parameters: map[string]string{},
	}

	for _, arg := range c.Args {
		parts := strings.SplitN(arg, "=", 2)

		if len(parts) != 2 {
			return fmt.Errorf("Key=Value expected: %s", arg)
		}

		opts.Parameters[parts[0]] = parts[1]
	}

	c.Startf("Updating parameters")

	if err := provider(c).SystemUpdate(opts); err != nil {
		return err
	}

	if c.Bool("wait") {
		if err := waitForRackRunning(c); err != nil {
			return err
		}
	}

	return c.OK()
}

func RackPs(c *stdcli.Context) error {
	var opts structs.SystemProcessesOptions

	if err := c.Options(&opts); err != nil {
		return err
	}

	ps, err := provider(c).SystemProcesses(opts)
	if err != nil {
		return err
	}

	t := c.Table("ID", "APP", "NAME", "RELEASE", "STARTED", "COMMAND")

	for _, p := range ps {
		t.AddRow(p.Id, p.App, p.Name, p.Release, helpers.Ago(p.Started), p.Command)
	}

	return t.Print()
}

func RackReleases(c *stdcli.Context) error {
	rs, err := provider(c).SystemReleases()
	if err != nil {
		return err
	}

	t := c.Table("VERSION", "UPDATED")

	for _, r := range rs {
		t.AddRow(r.Id, helpers.Ago(r.Created))
	}

	return t.Print()
}

func RackScale(c *stdcli.Context) error {
	s, err := provider(c).SystemGet()
	if err != nil {
		return err
	}

	var opts structs.SystemUpdateOptions
	update := false

	if v, ok := c.Value("count").(int); ok {
		opts.Count = options.Int(v)
		update = true
	}

	if v, ok := c.Value("type").(string); ok {
		opts.Type = options.String(v)
		update = true
	}

	if update {
		c.Startf("Scaling rack")

		if err := provider(c).SystemUpdate(opts); err != nil {
			return err
		}

		return c.OK()
	}

	i := c.Info()

	i.Add("Autoscale", s.Parameters["Autoscale"])
	i.Add("Count", fmt.Sprintf("%d", s.Count))
	i.Add("Status", s.Status)
	i.Add("Type", s.Type)

	return i.Print()
}

func RackStart(c *stdcli.Context) error {
	name := coalesce(c.String("name"), "convox")
	router := coalesce(c.String("router"), "10.42.0.0")

	cmd, err := rackCommand(name, version, router)
	if err != nil {
		return err
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	go handleSignalTermination(name)

	return cmd.Run()
}

func RackUpdate(c *stdcli.Context) error {
	s, err := provider(c).SystemGet()
	if err != nil {
		return err
	}

	// get next available version taking required releases into account
	target, err := cv.Next(s.Version)
	if err != nil {
		return err
	}

	if v := c.Arg(0); v != "" && v < target {
		target = v
	}

	c.Startf("Updating to <release>%s</release>", target)

	if err := provider(c).SystemUpdate(structs.SystemUpdateOptions{Version: options.String(target)}); err != nil {
		return err
	}

	if c.Bool("wait") {
		if err := waitForRackRunning(c); err != nil {
			return err
		}
	}

	return c.OK()
}
