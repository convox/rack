package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/convox/rack/cmd/convox/stdcli"
	"gopkg.in/urfave/cli.v1"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "apps",
		Action:      cmdApps,
		Description: "list deployed apps",
		Flags:       []cli.Flag{rackFlag},
		Subcommands: []cli.Command{
			{
				Name:        "cancel",
				Description: "cancel an update",
				Usage:       "",
				Action:      cmdAppCancel,
				Flags:       []cli.Flag{appFlag, rackFlag},
			},
			{
				Name:        "create",
				Description: "create a new application",
				Usage:       "<name>",
				Action:      cmdAppCreate,
				Flags: []cli.Flag{
					rackFlag,
					cli.BoolFlag{
						Name:   "wait",
						EnvVar: "CONVOX_WAIT",
						Usage:  "wait for app to finish creating before returning",
					},
				},
			},
			{
				Name:        "delete",
				Description: "delete an application",
				Usage:       "<name>",
				Action:      cmdAppDelete,
				Flags:       []cli.Flag{rackFlag},
			},
			{
				Name:        "info",
				Description: "see info about an app",
				Usage:       "[name]",
				Action:      cmdAppInfo,
				Flags:       []cli.Flag{appFlag, rackFlag},
			},
			{
				Name:        "params",
				Description: "list advanced parameters for an app",
				Usage:       "[name]",
				Action:      cmdAppParams,
				Flags:       []cli.Flag{appFlag, rackFlag},
				Subcommands: []cli.Command{
					{
						Name:        "set",
						Description: "update advanced parameters for an app",
						Usage:       "NAME=VALUE [NAME=VALUE]",
						Action:      cmdAppParamsSet,
						Flags:       []cli.Flag{appFlag, rackFlag},
					},
				},
			},
		},
	})
}

func cmdApps(c *cli.Context) error {
	if len(c.Args()) > 0 {
		return stdcli.Error(fmt.Errorf("`convox apps` does not take arguments. Perhaps you meant `convox apps create`?"))
	}

	if c.Bool("help") {
		stdcli.Usage(c, "")
		return nil
	}

	apps, err := rackClient(c).GetApps()
	if err != nil {
		return stdcli.Error(err)
	}

	t := stdcli.NewTable("APP", "STATUS")

	for _, app := range apps {
		t.AddRow(app.Name, app.Status)
	}

	t.Print()
	return nil
}

func cmdAppCancel(c *cli.Context) error {
	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.Error(err)
	}

	if app == "" {
		return stdcli.Error(fmt.Errorf("must specify an app name"))
	}

	stdcli.Startf("Cancelling update for <app>%s</app>", app)

	if err := rackClient(c).CancelApp(app); err != nil {
		return stdcli.Error(err)
	}

	stdcli.Wait("CANCELLED")

	return nil
}

func cmdAppCreate(c *cli.Context) error {
	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.Error(err)
	}

	if len(c.Args()) > 0 {
		app = c.Args()[0]
	}

	if app == "" {
		return stdcli.Error(fmt.Errorf("must specify an app name"))
	}

	stdcli.Startf("Creating app <app>%s</app>", app)

	_, err = rackClient(c).CreateApp(app)
	if err != nil {
		return stdcli.Error(err)
	}

	stdcli.Wait("CREATING")

	if c.Bool("wait") {
		stdcli.Startf("Waiting for <app>%s</app>", app)

		if err := waitForAppRunning(c, app); err != nil {
			stdcli.Error(err)
		}

		stdcli.OK()
	}

	return nil
}

func cmdAppDelete(c *cli.Context) error {
	if len(c.Args()) < 1 {
		stdcli.Usage(c, "delete")
		return nil
	}

	app := c.Args()[0]

	stdcli.Startf("Deleting <app>%s</app>", app)

	_, err := rackClient(c).DeleteApp(app)
	if err != nil {
		return stdcli.Error(err)
	}

	stdcli.Wait("DELETING")

	return nil
}

func cmdAppInfo(c *cli.Context) error {
	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.Error(err)
	}

	if len(c.Args()) > 0 {
		app = c.Args()[0]
	}

	a, err := rackClient(c).GetApp(app)
	if err != nil {
		return stdcli.Error(err)
	}

	formation, err := rackClient(c).ListFormation(app)
	if err != nil {
		return stdcli.Error(err)
	}

	ps := make([]string, len(formation))
	endpoints := []string{}

	for i, f := range formation {
		ps[i] = f.Name

		for _, port := range f.Ports {
			endpoints = append(endpoints, fmt.Sprintf("%s:%d (%s)", f.Balancer, port, f.Name))
		}
	}

	sort.Strings(ps)

	info := stdcli.NewInfo()

	info.Add("Name", a.Name)
	info.Add("Status", a.Status)
	info.Add("Release", stdcli.Default(a.Release, "(none)"))
	info.Add("Processes", stdcli.Default(strings.Join(ps, " "), "(none)"))
	info.Add("Endpoints", strings.Join(endpoints, "\n           "))

	info.Print()

	return nil
}

func cmdAppParams(c *cli.Context) error {
	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.Error(err)
	}

	params, err := rackClient(c).ListParameters(app)
	if err != nil {
		return stdcli.Error(err)
	}

	keys := []string{}

	for key := range params {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	t := stdcli.NewTable("NAME", "VALUE")

	for _, key := range keys {
		t.AddRow(key, params[key])
	}

	t.Print()
	return nil
}

func cmdAppParamsSet(c *cli.Context) error {
	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.Error(err)
	}

	params := map[string]string{}

	for _, arg := range c.Args() {
		parts := strings.SplitN(arg, "=", 2)

		if len(parts) != 2 {
			return stdcli.Error(fmt.Errorf("invalid argument: %s", arg))
		}

		params[parts[0]] = parts[1]
	}

	stdcli.Startf("Updating parameters")

	err = rackClient(c).SetParameters(app, params)
	if err != nil {
		return stdcli.Error(err)
	}

	stdcli.OK()

	return nil
}

func waitForAppRunning(c *cli.Context, app string) error {
	timeout := time.After(30 * time.Minute)
	tick := time.Tick(5 * time.Second)

	failed := false

	for {
		select {
		case <-tick:
			a, err := rackClient(c).GetApp(app)
			if err != nil {
				return err
			}

			switch a.Status {
			case "failed", "running":
				if failed {
					stdcli.Writef("<ok>DONE</ok>\n")
					return fmt.Errorf("Update rolled back")
				}
				return nil
			case "rollback":
				if !failed {
					failed = true
					stdcli.Writef("<fail>FAILED</fail>\n")
					stdcli.Startf("Rolling back")
				}
			}
		case <-timeout:
			return fmt.Errorf("timeout")
		}
	}

	return nil
}
