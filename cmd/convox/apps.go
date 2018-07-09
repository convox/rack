package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/convox/rack/options"
	"github.com/convox/rack/structs"
	"github.com/convox/stdcli"
)

func init() {
	CLI.Command("apps", "list apps", Apps, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack},
		Validate: stdcli.Args(0),
	})

	CLI.Command("apps cancel", "cancel an app update", AppsCancel, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack, flagApp},
		Usage:    "[app]",
		Validate: stdcli.ArgsMax(1),
	})

	CLI.Command("apps create", "create an app", AppsCreate, stdcli.CommandOptions{
		Flags:    append(stdcli.OptionFlags(structs.AppCreateOptions{}), flagRack, flagWait),
		Usage:    "<app>",
		Validate: stdcli.Args(1),
	})

	CLI.Command("apps delete", "delete an app", AppsDelete, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack, flagWait},
		Usage:    "<app>",
		Validate: stdcli.Args(1),
	})

	CLI.Command("apps info", "get information about an app", AppsInfo, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagApp, flagRack},
		Usage:    "[app]",
		Validate: stdcli.ArgsMax(1),
	})

	CLI.Command("apps params", "display app parameters", AppsParams, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagApp, flagRack},
		Usage:    "[app]",
		Validate: stdcli.ArgsMax(1),
	})

	CLI.Command("apps params set", "set app parameters", AppsParamsSet, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagApp, flagRack, flagWait},
		Usage:    "<Key=Value> [Key=Value]...",
		Validate: stdcli.ArgsMin(1),
	})

	CLI.Command("apps sleep", "sleep an app", AppsSleep, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagApp, flagRack},
		Usage:    "[app]",
		Validate: stdcli.ArgsMax(1),
	})

	CLI.Command("apps wake", "wake an app", AppsWake, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagApp, flagRack},
		Usage:    "[app]",
		Validate: stdcli.ArgsMax(1),
	})
}

func Apps(c *stdcli.Context) error {
	as, err := provider(c).AppList()
	if err != nil {
		return err
	}

	t := c.Table("APP", "STATUS", "GEN", "RELEASE")

	for _, a := range as {
		t.AddRow(a.Name, a.Status, a.Generation, a.Release)
	}

	return t.Print()
}

func AppsCancel(c *stdcli.Context) error {
	c.Startf("Cancelling <app>%s</app>", app(c))

	if err := provider(c).AppCancel(app(c)); err != nil {
		return err
	}

	return c.OK()
}

func AppsCreate(c *stdcli.Context) error {
	app := c.Args[0]

	var opts structs.AppCreateOptions

	if err := c.Options(&opts); err != nil {
		return err
	}

	c.Startf("Creating <app>%s</app>", app)

	if _, err := provider(c).AppCreate(app, opts); err != nil {
		return err
	}

	if c.Bool("wait") {
		if err := waitForAppRunning(c, app); err != nil {
			return err
		}
	}

	return c.OK()
}

func AppsDelete(c *stdcli.Context) error {
	app := c.Args[0]

	c.Startf("Deleting <app>%s</app>", app)

	if err := provider(c).AppDelete(app); err != nil {
		return err
	}

	if c.Bool("wait") {
		if err := waitForAppDeleted(c, app); err != nil {
			return err
		}
	}

	return c.OK()
}

func AppsInfo(c *stdcli.Context) error {
	a, err := provider(c).AppGet(coalesce(c.Arg(0), app(c)))
	if err != nil {
		return err
	}

	i := c.Info()

	i.Add("Name", a.Name)
	i.Add("Status", a.Status)
	i.Add("Gen", a.Generation)
	i.Add("Release", a.Release)

	return i.Print()
}

func AppsParams(c *stdcli.Context) error {
	a, err := provider(c).AppGet(coalesce(c.Arg(0), app(c)))
	if err != nil {
		return err
	}

	keys := []string{}

	for k := range a.Parameters {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	i := c.Info()

	for _, k := range keys {
		i.Add(k, a.Parameters[k])
	}

	return i.Print()
}

func AppsParamsSet(c *stdcli.Context) error {
	opts := structs.AppUpdateOptions{
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

	if err := provider(c).AppUpdate(app(c), opts); err != nil {
		return err
	}

	if c.Bool("wait") {
		if err := waitForAppWithLogs(c, app(c)); err != nil {
			return err
		}
	}

	return c.OK()
}

func AppsSleep(c *stdcli.Context) error {
	app := coalesce(c.Arg(0), app(c))

	c.Startf("Sleeping <app>%s</app>", app)

	if err := provider(c).AppUpdate(app, structs.AppUpdateOptions{Sleep: options.Bool(true)}); err != nil {
		return err
	}

	return c.OK()
}

func AppsWake(c *stdcli.Context) error {
	app := coalesce(c.Arg(0), app(c))

	c.Startf("Sleeping <app>%s</app>", app)

	if err := provider(c).AppUpdate(app, structs.AppUpdateOptions{Sleep: options.Bool(false)}); err != nil {
		return err
	}

	return c.OK()
}
