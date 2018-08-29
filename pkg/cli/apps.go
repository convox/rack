package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/sdk"
	"github.com/convox/stdcli"
)

func init() {
	register("apps", "list apps", Apps, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack},
		Validate: stdcli.Args(0),
	})

	register("apps cancel", "cancel an app update", AppsCancel, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack, flagApp},
		Usage:    "[app]",
		Validate: stdcli.ArgsMax(1),
	})

	register("apps create", "create an app", AppsCreate, stdcli.CommandOptions{
		Flags:    append(stdcli.OptionFlags(structs.AppCreateOptions{}), flagRack, flagWait),
		Usage:    "<app>",
		Validate: stdcli.ArgsMax(1),
	})

	register("apps delete", "delete an app", AppsDelete, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack, flagWait},
		Usage:    "<app>",
		Validate: stdcli.Args(1),
	})

	register("apps info", "get information about an app", AppsInfo, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagApp, flagRack},
		Usage:    "[app]",
		Validate: stdcli.ArgsMax(1),
	})

	register("apps params", "display app parameters", AppsParams, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagApp, flagRack},
		Usage:    "[app]",
		Validate: stdcli.ArgsMax(1),
	})

	register("apps params set", "set app parameters", AppsParamsSet, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagApp, flagRack, flagWait},
		Usage:    "<Key=Value> [Key=Value]...",
		Validate: stdcli.ArgsMin(1),
	})

	register("apps sleep", "sleep an app", AppsSleep, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagApp, flagRack},
		Usage:    "[app]",
		Validate: stdcli.ArgsMax(1),
	})

	register("apps wake", "wake an app", AppsWake, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagApp, flagRack},
		Usage:    "[app]",
		Validate: stdcli.ArgsMax(1),
	})

	register("apps wait", "wait for an app to finish updating", AppsWait, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagApp, flagRack},
		Usage:    "[app]",
		Validate: stdcli.ArgsMax(1),
	})
}

func Apps(rack sdk.Interface, c *stdcli.Context) error {
	as, err := rack.AppList()
	if err != nil {
		return err
	}

	t := c.Table("APP", "STATUS", "GEN", "RELEASE")

	for _, a := range as {
		t.AddRow(a.Name, a.Status, a.Generation, a.Release)
	}

	return t.Print()
}

func AppsCancel(rack sdk.Interface, c *stdcli.Context) error {
	c.Startf("Cancelling <app>%s</app>", app(c))

	if err := rack.AppCancel(app(c)); err != nil {
		return err
	}

	return c.OK()
}

func AppsCreate(rack sdk.Interface, c *stdcli.Context) error {
	app := coalesce(c.Arg(0), app(c))

	var opts structs.AppCreateOptions

	if err := c.Options(&opts); err != nil {
		return err
	}

	c.Startf("Creating <app>%s</app>", app)

	if _, err := rack.AppCreate(app, opts); err != nil {
		return err
	}

	if c.Bool("wait") {
		if err := waitForAppRunning(rack, c, app); err != nil {
			return err
		}
	}

	return c.OK()
}

func AppsDelete(rack sdk.Interface, c *stdcli.Context) error {
	app := c.Args[0]

	c.Startf("Deleting <app>%s</app>", app)

	if err := rack.AppDelete(app); err != nil {
		return err
	}

	if c.Bool("wait") {
		if err := waitForAppDeleted(rack, c, app); err != nil {
			return err
		}
	}

	return c.OK()
}

func AppsInfo(rack sdk.Interface, c *stdcli.Context) error {
	a, err := rack.AppGet(coalesce(c.Arg(0), app(c)))
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

func AppsParams(rack sdk.Interface, c *stdcli.Context) error {
	s, err := rack.SystemGet()
	if err != nil {
		return err
	}

	var params map[string]string

	app := coalesce(c.Arg(0), app(c))

	if s.Version <= "20180708231844" {
		params, err = rack.AppParametersGet(app)
		if err != nil {
			return err
		}
	} else {
		a, err := rack.AppGet(app)
		if err != nil {
			return err
		}
		params = a.Parameters
	}

	keys := []string{}

	for k := range params {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	i := c.Info()

	for _, k := range keys {
		i.Add(k, params[k])
	}

	return i.Print()
}

func AppsParamsSet(rack sdk.Interface, c *stdcli.Context) error {
	s, err := rack.SystemGet()
	if err != nil {
		return err
	}

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

	if s.Version <= "20180708231844" {
		if err := rack.AppParametersSet(app(c), opts.Parameters); err != nil {
			return err
		}
	} else {
		if err := rack.AppUpdate(app(c), opts); err != nil {
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

func AppsSleep(rack sdk.Interface, c *stdcli.Context) error {
	app := coalesce(c.Arg(0), app(c))

	c.Startf("Sleeping <app>%s</app>", app)

	if err := rack.AppUpdate(app, structs.AppUpdateOptions{Sleep: options.Bool(true)}); err != nil {
		return err
	}

	return c.OK()
}

func AppsWake(rack sdk.Interface, c *stdcli.Context) error {
	app := coalesce(c.Arg(0), app(c))

	c.Startf("Waking <app>%s</app>", app)

	if err := rack.AppUpdate(app, structs.AppUpdateOptions{Sleep: options.Bool(false)}); err != nil {
		return err
	}

	return c.OK()
}

func AppsWait(rack sdk.Interface, c *stdcli.Context) error {
	app := coalesce(c.Arg(0), app(c))

	c.Startf("Waiting for app")

	if err := waitForAppWithLogs(rack, c, app); err != nil {
		return err
	}

	return c.OK()
}
