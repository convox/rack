package cli

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/sdk"
	"github.com/convox/stdcli"
)

func init() {
	register("resources", "list resources", Resources, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack, flagApp},
		Validate: stdcli.Args(0),
	})

	register("resources info", "get information about a resource", ResourcesInfo, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack, flagApp},
		Usage:    "<resource>",
		Validate: stdcli.Args(1),
	})

	register("resources proxy", "proxy a local port to a resource", ResourcesProxy, stdcli.CommandOptions{
		Flags: []stdcli.Flag{
			flagRack,
			flagApp,
			stdcli.IntFlag("port", "p", "local port"),
			stdcli.BoolFlag("tls", "t", "wrap connection in tls"),
		},
		Usage:    "<resource>",
		Validate: stdcli.Args(1),
	})

	register("resources url", "get url for a resource", ResourcesUrl, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack, flagApp},
		Usage:    "<resource>",
		Validate: stdcli.Args(1),
	})

	register("rack resources", "list resources", RackResources, stdcli.CommandOptions{
		Flags:     []stdcli.Flag{flagRack},
		Invisible: true,
		Validate:  stdcli.Args(0),
	})

	register("rack resources create", "create a resource", RackResourcesCreate, stdcli.CommandOptions{
		Flags: []stdcli.Flag{
			flagRack,
			flagWait,
			stdcli.StringFlag("name", "n", "resource name"),
		},
		Invisible: true,
		Usage:     "<type> [Option=Value]...",
		Validate:  stdcli.ArgsMin(1),
	})

	register("rack resources delete", "delete a resource", RackResourcesDelete, stdcli.CommandOptions{
		Flags:     []stdcli.Flag{flagRack, flagWait},
		Invisible: true,
		Usage:     "<name>",
		Validate:  stdcli.Args(1),
	})

	register("rack resources info", "get information about a resource", RackResourcesInfo, stdcli.CommandOptions{
		Flags:     []stdcli.Flag{flagRack},
		Invisible: true,
		Usage:     "<resource>",
		Validate:  stdcli.Args(1),
	})

	register("rack resources link", "link a resource to an app", RackResourcesLink, stdcli.CommandOptions{
		Flags:     []stdcli.Flag{flagApp, flagRack, flagWait},
		Invisible: true,
		Usage:     "<resource>",
		Validate:  stdcli.Args(1),
	})

	register("rack resources options", "list options for a resource type", RackResourcesOptions, stdcli.CommandOptions{
		Flags:     []stdcli.Flag{flagRack},
		Invisible: true,
		Usage:     "<resource>",
		Validate:  stdcli.Args(1),
	})

	register("rack resources proxy", "proxy a local port to a rack resource", RackResourcesProxy, stdcli.CommandOptions{
		Flags: []stdcli.Flag{
			flagRack,
			stdcli.IntFlag("port", "p", "local port"),
			stdcli.BoolFlag("tls", "t", "wrap connection in tls"),
		},
		Invisible: true,
		Usage:     "<resource>",
		Validate:  stdcli.Args(1),
	})

	register("rack resources types", "list resource types", RackResourcesTypes, stdcli.CommandOptions{
		Flags:     []stdcli.Flag{flagRack},
		Invisible: true,
		Validate:  stdcli.Args(0),
	})

	register("rack resources update", "update resource options", RackResourcesUpdate, stdcli.CommandOptions{
		Flags:     []stdcli.Flag{flagRack, flagWait},
		Invisible: true,
		Usage:     "<name> [Option=Value]...",
		Validate:  stdcli.ArgsMin(1),
	})

	register("rack resources unlink", "unlink a resource from an app", RackResourcesUnlink, stdcli.CommandOptions{
		Flags:     []stdcli.Flag{flagApp, flagRack, flagWait},
		Invisible: true,
		Usage:     "<resource>",
		Validate:  stdcli.Args(1),
	})

	register("rack resources url", "get url for a resource", RackResourcesUrl, stdcli.CommandOptions{
		Flags:     []stdcli.Flag{flagRack},
		Invisible: true,
		Usage:     "<resource>",
		Validate:  stdcli.Args(1),
	})
}

func Resources(rack sdk.Interface, c *stdcli.Context) error {
	s, err := rack.SystemGet()
	if err != nil {
		return err
	}

	if s.Version <= "20190111211123" {
		return fmt.Errorf("command unavailable, please upgrade this rack")
	}

	rs, err := rack.ResourceList(app(c))
	if err != nil {
		return err
	}

	t := c.Table("NAME", "TYPE", "URL")

	for _, r := range rs {
		t.AddRow(r.Name, r.Type, r.Url)
	}

	return t.Print()
}

func ResourcesInfo(rack sdk.Interface, c *stdcli.Context) error {
	s, err := rack.SystemGet()
	if err != nil {
		return err
	}

	if s.Version <= "20190111211123" {
		return fmt.Errorf("command unavailable, please upgrade this rack")
	}

	r, err := rack.ResourceGet(app(c), c.Arg(0))
	if err != nil {
		return err
	}

	i := c.Info()

	i.Add("Name", r.Name)
	i.Add("Type", r.Type)

	if r.Url != "" {
		i.Add("URL", r.Url)
	}

	return i.Print()
}

func ResourcesProxy(rack sdk.Interface, c *stdcli.Context) error {
	s, err := rack.SystemGet()
	if err != nil {
		return err
	}

	if s.Version <= "20190111211123" {
		return fmt.Errorf("command unavailable, please upgrade this rack")
	}

	r, err := rack.ResourceGet(app(c), c.Arg(0))
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

	go proxy(rack, c, port, remotehost, rpi, c.Bool("tls"))

	<-c.Done()

	return nil
}

func ResourcesUrl(rack sdk.Interface, c *stdcli.Context) error {
	s, err := rack.SystemGet()
	if err != nil {
		return err
	}

	if s.Version <= "20190111211123" {
		return fmt.Errorf("command unavailable, please upgrade this rack")
	}

	r, err := rack.ResourceGet(app(c), c.Arg(0))
	if err != nil {
		return err
	}

	if r.Url == "" {
		return fmt.Errorf("no url for resource: %s", r.Name)
	}

	fmt.Fprintf(c, "%s\n", r.Url)

	return nil
}

func RackResources(rack sdk.Interface, c *stdcli.Context) error {
	s, err := rack.SystemGet()
	if err != nil {
		return err
	}

	var rs structs.Resources

	if s.Version <= "20190111211123" {
		rs, err = rack.SystemResourceListClassic()
	} else {
		rs, err = rack.SystemResourceList()
	}
	if err != nil {
		return err
	}

	t := c.Table("NAME", "TYPE", "STATUS")

	for _, r := range rs {
		t.AddRow(r.Name, r.Type, r.Status)
	}

	return t.Print()
}

func RackResourcesCreate(rack sdk.Interface, c *stdcli.Context) error {
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

	s, err := rack.SystemGet()
	if err != nil {
		return err
	}

	var r *structs.Resource

	if s.Version <= "20180708231844" {
		r, err = rack.ResourceCreateClassic(c.Arg(0), opts)
	} else if s.Version <= "20190111211123" {
		r, err = rack.SystemResourceCreateClassic(c.Arg(0), opts)
	} else {
		r, err = rack.SystemResourceCreate(c.Arg(0), opts)
	}
	if err != nil {
		return err
	}

	if c.Bool("wait") {
		if err := waitForResourceRunning(rack, c, r.Name); err != nil {
			return err
		}
	}

	return c.OK(r.Name)
}

func RackResourcesDelete(rack sdk.Interface, c *stdcli.Context) error {
	s, err := rack.SystemGet()
	if err != nil {
		return err
	}

	c.Startf("Deleting resource")

	if s.Version <= "20190111211123" {
		err = rack.SystemResourceDeleteClassic(c.Arg(0))
	} else {
		err = rack.SystemResourceDelete(c.Arg(0))
	}
	if err != nil {
		return err
	}

	if c.Bool("wait") {
		if err := waitForResourceDeleted(rack, c, c.Arg(0)); err != nil {
			return err
		}
	}

	return c.OK()
}

func RackResourcesInfo(rack sdk.Interface, c *stdcli.Context) error {
	s, err := rack.SystemGet()
	if err != nil {
		return err
	}

	var r *structs.Resource

	if s.Version <= "20190111211123" {
		r, err = rack.SystemResourceGetClassic(c.Arg(0))
	} else {
		r, err = rack.SystemResourceGet(c.Arg(0))
	}
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

func RackResourcesLink(rack sdk.Interface, c *stdcli.Context) error {
	s, err := rack.SystemGet()
	if err != nil {
		return err
	}

	c.Startf("Linking to <app>%s</app>", app(c))

	resource := c.Arg(0)

	if s.Version <= "20190111211123" {
		_, err = rack.SystemResourceLinkClassic(resource, app(c))
	} else {
		_, err = rack.SystemResourceLink(resource, app(c))
	}
	if err != nil {
		return err
	}

	if c.Bool("wait") {
		if err := waitForResourceRunning(rack, c, resource); err != nil {
			return err
		}
	}

	return c.OK()
}

func RackResourcesOptions(rack sdk.Interface, c *stdcli.Context) error {
	s, err := rack.SystemGet()
	if err != nil {
		return err
	}

	var rts structs.ResourceTypes

	if s.Version <= "20190111211123" {
		rts, err = rack.SystemResourceTypesClassic()
	} else {
		rts, err = rack.SystemResourceTypes()
	}
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

func RackResourcesProxy(rack sdk.Interface, c *stdcli.Context) error {
	s, err := rack.SystemGet()
	if err != nil {
		return err
	}

	var r *structs.Resource

	if s.Version <= "20190111211123" {
		r, err = rack.SystemResourceGetClassic(c.Arg(0))
	} else {
		r, err = rack.SystemResourceGet(c.Arg(0))
	}
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

	go proxy(rack, c, port, remotehost, rpi, c.Bool("tls"))

	<-c.Done()

	return nil
}

func RackResourcesTypes(rack sdk.Interface, c *stdcli.Context) error {
	s, err := rack.SystemGet()
	if err != nil {
		return err
	}

	var rts structs.ResourceTypes

	if s.Version <= "20190111211123" {
		rts, err = rack.SystemResourceTypesClassic()
	} else {
		rts, err = rack.SystemResourceTypes()
	}
	if err != nil {
		return err
	}

	t := c.Table("TYPE")

	for _, rt := range rts {
		t.AddRow(rt.Name)
	}

	return t.Print()
}

func RackResourcesUnlink(rack sdk.Interface, c *stdcli.Context) error {
	s, err := rack.SystemGet()
	if err != nil {
		return err
	}

	c.Startf("Unlinking from <app>%s</app>", app(c))

	resource := c.Arg(0)

	if s.Version <= "20190111211123" {
		_, err = rack.SystemResourceUnlinkClassic(resource, app(c))
	} else {
		_, err = rack.SystemResourceUnlink(resource, app(c))
	}
	if err != nil {
		return err
	}

	if c.Bool("wait") {
		if err := waitForResourceRunning(rack, c, resource); err != nil {
			return err
		}
	}

	return c.OK()
}

func RackResourcesUpdate(rack sdk.Interface, c *stdcli.Context) error {
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

	s, err := rack.SystemGet()
	if err != nil {
		return err
	}

	resource := c.Arg(0)

	if s.Version <= "20180708231844" {
		_, err = rack.ResourceUpdateClassic(resource, opts)
	} else if s.Version <= "20190111211123" {
		_, err = rack.SystemResourceUpdateClassic(resource, opts)
	} else {
		_, err = rack.SystemResourceUpdate(resource, opts)
	}
	if err != nil {
		return err
	}

	if c.Bool("wait") {
		if err := waitForResourceRunning(rack, c, resource); err != nil {
			return err
		}
	}

	return c.OK()
}

func RackResourcesUrl(rack sdk.Interface, c *stdcli.Context) error {
	s, err := rack.SystemGet()
	if err != nil {
		return err
	}

	var r *structs.Resource

	if s.Version <= "20190111211123" {
		r, err = rack.SystemResourceGetClassic(c.Arg(0))
	} else {
		r, err = rack.SystemResourceGet(c.Arg(0))
	}
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
