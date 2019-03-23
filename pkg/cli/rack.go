package cli

import (
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"

	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/sdk"
	"github.com/convox/stdcli"

	pv "github.com/convox/rack/provider"
	cv "github.com/convox/version"
)

func init() {
	register("rack", "get information about the rack", Rack, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack},
		Validate: stdcli.Args(0),
	})

	registerWithoutProvider("rack install", "install a rack", RackInstall, stdcli.CommandOptions{
		Flags:    append(stdcli.OptionFlags(structs.SystemInstallOptions{})),
		Usage:    "<type> [Parameter=Value]...",
		Validate: stdcli.ArgsMin(1),
	})

	register("rack logs", "get logs for the rack", RackLogs, stdcli.CommandOptions{
		Flags:    append(stdcli.OptionFlags(structs.LogsOptions{}), flagNoFollow, flagRack),
		Validate: stdcli.Args(0),
	})

	register("rack params", "display rack parameters", RackParams, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack},
		Validate: stdcli.Args(0),
	})

	register("rack params set", "set rack parameters", RackParamsSet, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack, flagWait},
		Usage:    "<Key=Value> [Key=Value]...",
		Validate: stdcli.ArgsMin(1),
	})

	register("rack ps", "list rack processes", RackPs, stdcli.CommandOptions{
		Flags:    append(stdcli.OptionFlags(structs.SystemProcessesOptions{}), flagRack),
		Validate: stdcli.Args(0),
	})

	register("rack releases", "list rack version history", RackReleases, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack},
		Validate: stdcli.Args(0),
	})

	register("rack scale", "scale the rack", RackScale, stdcli.CommandOptions{
		Flags: []stdcli.Flag{
			flagRack,
			stdcli.IntFlag("count", "c", "instance count"),
			stdcli.StringFlag("type", "t", "instance type"),
		},
		Validate: stdcli.Args(0),
	})

	registerWithoutProvider("rack start", "start local rack", RackStart, stdcli.CommandOptions{
		Flags: []stdcli.Flag{
			stdcli.StringFlag("id", "", "identifier"),
			stdcli.StringFlag("name", "n", "rack name"),
			stdcli.StringFlag("router", "r", "router host"),
		},
		Invisible: true,
		Validate:  stdcli.Args(0),
	})

	registerWithoutProvider("rack uninstall", "uninstall a rack", RackUninstall, stdcli.CommandOptions{
		Flags:    append(stdcli.OptionFlags(structs.SystemUninstallOptions{})),
		Usage:    "<type> <name>",
		Validate: stdcli.Args(2),
	})

	register("rack update", "update the rack", RackUpdate, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack, flagWait},
		Validate: stdcli.ArgsMax(1),
	})

	register("rack wait", "wait for rack to finish updating", RackWait, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack},
		Validate: stdcli.Args(0),
	})
}

func Rack(rack sdk.Interface, c *stdcli.Context) error {
	s, err := rack.SystemGet()
	if err != nil {
		return err
	}

	i := c.Info()

	i.Add("Name", s.Name)
	i.Add("Provider", s.Provider)

	if s.Region != "" {
		i.Add("Region", s.Region)
	}

	if s.Domain != "" {
		if ri := s.Outputs["DomainInternal"]; ri != "" {
			i.Add("Router", fmt.Sprintf("%s (external)\n%s (internal)", s.Domain, ri))
		} else {
			i.Add("Router", s.Domain)
		}
	}

	i.Add("Status", s.Status)
	i.Add("Version", s.Version)

	return i.Print()
}

func RackInstall(rack sdk.Interface, c *stdcli.Context) error {
	var opts structs.SystemInstallOptions

	if err := c.Options(&opts); err != nil {
		return err
	}

	if opts.Version == nil {
		v, err := cv.Latest()
		if err != nil {
			return err
		}
		opts.Version = options.String(v)
	}

	if id, _ := c.SettingRead("id"); id != "" {
		opts.Id = options.String(id)
	}

	opts.Parameters = map[string]string{}

	for _, arg := range c.Args[1:] {
		parts := strings.SplitN(arg, "=", 2)

		if len(parts) != 2 {
			return fmt.Errorf("Key=Value expected: %s", arg)
		}

		opts.Parameters[parts[0]] = parts[1]
	}

	p, err := pv.FromName(c.Arg(0))
	if err != nil {
		return err
	}

	if !helpers.DefaultBool(opts.Raw, false) {
		c.Writef("   ___ ___  _  _ _   __ __ _  __\n")
		c.Writef("  / __/ _ \\| \\| \\ \\ / / _ \\ \\/ /\n")
		c.Writef(" | (_| (_) |  ` |\\ V / (_) )  ( \n")
		c.Writef("  \\___\\___/|_|\\_| \\_/ \\___/_/\\_\\\n")
		c.Writef("\n")
	}

	ep, err := p.SystemInstall(c, opts)
	if err != nil {
		return err
	}

	u, err := url.Parse(ep)
	if err != nil {
		return err
	}

	password := ""

	if u.User != nil {
		if pw, ok := u.User.Password(); ok {
			password = pw
		}
	}

	if err := c.SettingWriteKey("auth", u.Host, password); err != nil {
		return err
	}

	if host, _ := c.SettingRead("host"); host == "" {
		c.SettingWrite("host", u.Host)
	}

	return nil
}

func RackLogs(rack sdk.Interface, c *stdcli.Context) error {
	var opts structs.LogsOptions

	if err := c.Options(&opts); err != nil {
		return err
	}

	if c.Bool("no-follow") {
		opts.Follow = options.Bool(false)
	}

	opts.Prefix = options.Bool(true)

	r, err := rack.SystemLogs(opts)
	if err != nil {
		return err
	}

	io.Copy(c, r)

	return nil
}

func RackParams(rack sdk.Interface, c *stdcli.Context) error {
	s, err := rack.SystemGet()
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

func RackParamsSet(rack sdk.Interface, c *stdcli.Context) error {
	s, err := rack.SystemGet()
	if err != nil {
		return err
	}

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

	if s.Version <= "20180708231844" {
		if err := rack.AppParametersSet(s.Name, opts.Parameters); err != nil {
			return err
		}
	} else {
		if err := rack.SystemUpdate(opts); err != nil {
			return err
		}
	}

	if c.Bool("wait") {
		c.Writef("\n")

		if err := helpers.WaitForRackWithLogs(rack, c); err != nil {
			return err
		}
	}

	return c.OK()
}

func RackPs(rack sdk.Interface, c *stdcli.Context) error {
	var opts structs.SystemProcessesOptions

	if err := c.Options(&opts); err != nil {
		return err
	}

	ps, err := rack.SystemProcesses(opts)
	if err != nil {
		return err
	}

	t := c.Table("ID", "APP", "SERVICE", "STATUS", "RELEASE", "STARTED", "COMMAND")

	for _, p := range ps {
		t.AddRow(p.Id, p.App, p.Name, p.Status, p.Release, helpers.Ago(p.Started), p.Command)
	}

	return t.Print()
}

func RackReleases(rack sdk.Interface, c *stdcli.Context) error {
	rs, err := rack.SystemReleases()
	if err != nil {
		return err
	}

	t := c.Table("VERSION", "UPDATED")

	for _, r := range rs {
		t.AddRow(r.Id, helpers.Ago(r.Created))
	}

	return t.Print()
}

func RackScale(rack sdk.Interface, c *stdcli.Context) error {
	s, err := rack.SystemGet()
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

		if err := rack.SystemUpdate(opts); err != nil {
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

func RackStart(rack sdk.Interface, c *stdcli.Context) error {
	id := c.String("id")
	name := coalesce(c.String("name"), "convox")
	router := coalesce(c.String("router"), "10.42.0.0")

	cmd, args, err := rackCommand(name, c.Version(), router, id)
	if err != nil {
		return err
	}

	go handleSignalTermination(c, name)

	c.Execute("docker", "rm", "-f", name)

	return c.Run(cmd, args...)
}

func RackUninstall(rack sdk.Interface, c *stdcli.Context) error {
	var opts structs.SystemUninstallOptions

	if err := c.Options(&opts); err != nil {
		return err
	}

	if c.Reader().IsTerminal() {
		opts.Input = c.Reader()
	} else {
		if !c.Bool("force") {
			return fmt.Errorf("must use --force for non-interactive uninstall")
		}
	}

	p, err := pv.FromName(c.Arg(0))
	if err != nil {
		return err
	}

	if err := p.SystemUninstall(c.Arg(1), c, opts); err != nil {
		return err
	}

	return nil
}

func RackUpdate(rack sdk.Interface, c *stdcli.Context) error {
	target := c.Arg(0)

	// if no version specified, find the next version
	if target == "" {
		s, err := rack.SystemGet()
		if err != nil {
			return err
		}

		if s.Version == "dev" {
			target = "dev"
		} else {
			v, err := cv.Next(s.Version)
			if err != nil {
				return err
			}

			target = v
		}
	}

	c.Startf("Updating to <release>%s</release>", target)

	if err := rack.SystemUpdate(structs.SystemUpdateOptions{Version: options.String(target)}); err != nil {
		return err
	}

	if c.Bool("wait") {
		c.Writef("\n")

		if err := helpers.WaitForRackWithLogs(rack, c); err != nil {
			return err
		}
	}

	return c.OK()
}

func RackWait(rack sdk.Interface, c *stdcli.Context) error {
	c.Startf("Waiting for rack")

	c.Writef("\n")

	if err := helpers.WaitForRackWithLogs(rack, c); err != nil {
		return err
	}

	return c.OK()
}
