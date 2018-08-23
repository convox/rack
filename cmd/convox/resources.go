package main

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/stdcli"
)

func init() {
	CLI.Command("resources", "list resources", Resources, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack},
		Validate: stdcli.Args(0),
	})

	CLI.Command("resources create", "create a resource", ResourcesCreate, stdcli.CommandOptions{
		Flags: []stdcli.Flag{
			flagRack,
			flagWait,
			stdcli.StringFlag("name", "n", "resource name"),
		},
		Usage:    "<type> [Option=Value]...",
		Validate: stdcli.ArgsMin(1),
	})

	CLI.Command("resources delete", "delete a resource", ResourcesDelete, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack, flagWait},
		Usage:    "<name>",
		Validate: stdcli.Args(1),
	})

	CLI.Command("resources info", "get information about a resource", ResourcesInfo, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack},
		Usage:    "<resource>",
		Validate: stdcli.Args(1),
	})

	CLI.Command("resources link", "link a resource to an app", ResourcesLink, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagApp, flagRack, flagWait},
		Usage:    "<resource>",
		Validate: stdcli.Args(1),
	})

	CLI.Command("resources options", "list options for a resource type", ResourcesOptions, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack},
		Usage:    "<resource>",
		Validate: stdcli.Args(1),
	})

	CLI.Command("resources proxy", "get information about a resource", ResourcesProxy, stdcli.CommandOptions{
		Flags: []stdcli.Flag{
			flagRack,
			stdcli.IntFlag("port", "p", "local port"),
		},
		Usage:    "<resource>",
		Validate: stdcli.Args(1),
	})

	CLI.Command("resources types", "list resource types", ResourcesTypes, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack},
		Validate: stdcli.Args(0),
	})

	CLI.Command("resources update", "update resource options", ResourcesUpdate, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack, flagWait},
		Usage:    "<name> [Option=Value]...",
		Validate: stdcli.ArgsMin(1),
	})

	CLI.Command("resources unlink", "unlink a resource from an app", ResourcesUnlink, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagApp, flagRack, flagWait},
		Usage:    "<resource>",
		Validate: stdcli.Args(1),
	})

	CLI.Command("resources url", "get url for a resource", ResourcesUrl, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack},
		Usage:    "<resource>",
		Validate: stdcli.Args(1),
	})
}

func Resources(c *stdcli.Context) error {
	rs, err := provider(c).ResourceList()
	if err != nil {
		return err
	}

	t := c.Table("NAME", "TYPE", "STATUS")

	for _, r := range rs {
		t.AddRow(r.Name, r.Type, r.Status)
	}

	return t.Print()
}

func ResourcesCreate(c *stdcli.Context) error {
	var opts structs.ResourceCreateOptions

	if err := c.Options(&opts); err != nil {
		return err
	}

	if v := c.String("name"); v != "" {
		opts.Name = options.String(v)
	}

	opts.Parameters = map[string]string{}

	for _, arg := range c.Args[1:] {
		parts := strings.SplitN(arg, "=", 2)

		if len(parts) != 2 {
			return fmt.Errorf("Name=Value expected: %s", arg)
		}

		opts.Parameters[parts[0]] = parts[1]
	}

	c.Startf("Creating resource")

	s, err := provider(c).SystemGet()
	if err != nil {
		return err
	}

	var r *structs.Resource

	if s.Version <= "20180708231844" {
		r, err = provider(c).ResourceCreateClassic(c.Arg(0), opts)
	} else {
		r, err = provider(c).ResourceCreate(c.Arg(0), opts)
	}
	if err != nil {
		return err
	}

	if c.Bool("wait") {
		if err := waitForResourceRunning(c, r.Name); err != nil {
			return err
		}
	}

	return c.OK(r.Name)
}

func ResourcesDelete(c *stdcli.Context) error {
	c.Startf("Deleting resource")

	if err := provider(c).ResourceDelete(c.Arg(0)); err != nil {
		return err
	}

	if c.Bool("wait") {
		if err := waitForResourceDeleted(c, c.Arg(0)); err != nil {
			return err
		}
	}

	return c.OK()
}

func ResourcesInfo(c *stdcli.Context) error {
	r, err := provider(c).ResourceGet(c.Arg(0))
	if err != nil {
		return err
	}

	// fmt.Printf("r = %+v\n", r)

	i := c.Info()

	apps := []string{}

	for _, a := range r.Apps {
		apps = append(apps, a.Name)
	}

	sort.Strings(apps)

	options := []string{}

	for k, v := range r.Parameters {
		options = append(options, fmt.Sprintf("%s=%s", k, v))
	}

	sort.Strings(options)

	i.Add("Name", r.Name)
	i.Add("Type", r.Type)
	i.Add("Status", r.Status)
	i.Add("Options", strings.Join(options, "\n"))

	if r.Url != "" {
		i.Add("URL", r.Url)
	}

	if len(apps) > 0 {
		i.Add("Apps", strings.Join(apps, ", "))
	}

	return i.Print()
}

func ResourcesLink(c *stdcli.Context) error {
	c.Startf("Linking to <app>%s</app>", app(c))

	resource := c.Arg(0)

	if _, err := provider(c).ResourceLink(resource, app(c)); err != nil {
		return err
	}

	if c.Bool("wait") {
		if err := waitForResourceRunning(c, resource); err != nil {
			return err
		}
	}

	return c.OK()
}

func ResourcesOptions(c *stdcli.Context) error {
	rts, err := provider(c).ResourceTypes()
	if err != nil {
		return err
	}

	var rt *structs.ResourceType

	for _, t := range rts {
		if t.Name == c.Arg(0) {
			rt = &t
			break
		}
	}

	if rt == nil {
		return fmt.Errorf("no such resource type: %s", c.Arg(0))
	}

	t := c.Table("NAME", "DEFAULT", "DESCRIPTION")

	sort.Slice(rt.Parameters, rt.Parameters.Less)

	for _, p := range rt.Parameters {
		t.AddRow(p.Name, p.Default, p.Description)
	}

	return t.Print()
}

func ResourcesProxy(c *stdcli.Context) error {
	r, err := provider(c).ResourceGet(c.Arg(0))
	if err != nil {
		return err
	}

	if r.Url == "" {
		return fmt.Errorf("no url for resource: %s", r.Name)
	}

	u, err := url.Parse(r.Url)
	if err != nil {
		return err
	}

	remotehost := u.Hostname()
	remoteport := u.Port()

	if remoteport == "" {
		switch u.Scheme {
		case "http":
			remoteport = "80"
		case "https":
			remoteport = "443"
		default:
			return fmt.Errorf("unknown port for url: %s", r.Url)
		}
	}

	rpi, err := strconv.Atoi(remoteport)
	if err != nil {
		return err
	}

	port := rpi

	if p := c.Int("port"); p != 0 {
		port = p
	}

	go proxy(c, port, remotehost, rpi)

	select {}
}

func ResourcesTypes(c *stdcli.Context) error {
	rts, err := provider(c).ResourceTypes()
	if err != nil {
		return err
	}

	t := c.Table("TYPE")

	for _, rt := range rts {
		t.AddRow(rt.Name)
	}

	return t.Print()
}

func ResourcesUnlink(c *stdcli.Context) error {
	c.Startf("Unlinking from <app>%s</app>", app(c))

	resource := c.Arg(0)

	if _, err := provider(c).ResourceUnlink(resource, app(c)); err != nil {
		return err
	}

	if c.Bool("wait") {
		if err := waitForResourceRunning(c, resource); err != nil {
			return err
		}
	}

	return c.OK()
}

func ResourcesUpdate(c *stdcli.Context) error {
	opts := structs.ResourceUpdateOptions{
		Parameters: map[string]string{},
	}

	for _, arg := range c.Args[1:] {
		parts := strings.SplitN(arg, "=", 2)

		if len(parts) != 2 {
			return fmt.Errorf("Key=Value expected: %s", arg)
		}

		opts.Parameters[parts[0]] = parts[1]
	}

	c.Startf("Updating resource")

	s, err := provider(c).SystemGet()
	if err != nil {
		return err
	}

	resource := c.Arg(0)

	if s.Version <= "20180708231844" {
		if _, err := provider(c).ResourceUpdateClassic(resource, opts); err != nil {
			return err
		}
	} else {
		if _, err := provider(c).ResourceUpdate(resource, opts); err != nil {
			return err
		}
	}

	if c.Bool("wait") {
		if err := waitForResourceRunning(c, resource); err != nil {
			return err
		}
	}

	return c.OK()
}

func ResourcesUrl(c *stdcli.Context) error {
	r, err := provider(c).ResourceGet(c.Arg(0))
	if err != nil {
		return err
	}

	s, err := provider(c).SystemGet()
	if err != nil {
		return err
	}

	if s.Version <= "20180708231844" {
		if u := r.Parameters["Url"]; u != "" {
			fmt.Fprintf(c, "%s\n", u)
			return nil
		}
	}

	if r.Url == "" {
		return fmt.Errorf("no url for resource: %s", r.Name)
	}

	fmt.Fprintf(c, "%s\n", r.Url)

	return nil
}
