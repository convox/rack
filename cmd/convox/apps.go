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
				Usage:       "[options]",
				ArgsUsage:   "",
				Action:      cmdAppCancel,
				Flags:       []cli.Flag{appFlag, rackFlag},
			},
			{
				Name:        "create",
				Description: "create a new application",
				Usage:       "[name] [options]",
				ArgsUsage:   "[name] (inferred from current directory if not specified)",
				Action:      cmdAppCreate,
				Flags: []cli.Flag{
					rackFlag,
					cli.StringFlag{
						Name:  "generation, g",
						Usage: "generation of app to create",
					},
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
				ArgsUsage:   "",
				Action:      cmdAppParams,
				Flags:       []cli.Flag{appFlag, rackFlag},
				Subcommands: []cli.Command{
					{
						Name:        "set",
						Description: "update advanced parameters for an app",
						Usage:       "NAME=VALUE [NAME=VALUE] ... [options]",
						ArgsUsage:   "NAME=VALUE",
						Action:      cmdAppParamsSet,
						Flags:       []cli.Flag{appFlag, rackFlag},
					},
				},
			},
		},
	})
}

func cmdApps(c *cli.Context) error {
	stdcli.NeedHelp(c)
	stdcli.NeedArg(c, 0)

	apps, err := rackClient(c).GetApps()
	if err != nil {
		return stdcli.Error(err)
	}

	if len(apps) == 0 {
		stdcli.Writef("no apps found, try creating one via `convox apps create`\n")
		return nil
	}

	t := stdcli.NewTable("APP", "GEN", "STATUS")

	for _, app := range apps {
		t.AddRow(app.Name, app.Generation, app.Status)
	}

	t.Print()
	return nil
}

func cmdAppCancel(c *cli.Context) error {
	stdcli.NeedHelp(c)
	stdcli.NeedArg(c, 0)

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
	stdcli.NeedHelp(c)

	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.Error(err)
	}

	if len(c.Args()) > 0 {
		// accept no more than 1 argument
		stdcli.NeedArg(c, 1)
		app = c.Args()[0]
	}

	if app == "" {
		return stdcli.Error(fmt.Errorf("must specify an app name"))
	}

	generation := c.String("generation")

	stdcli.Startf("Creating app <app>%s</app>", app)

	_, err = rackClient(c).CreateApp(app, generation)
	if err != nil {
		return stdcli.Error(err)
	}

	stdcli.Wait("CREATING")

	if c.Bool("wait") {
		stdcli.Startf("Waiting for <app>%s</app>", app)

		if err := waitForAppRunning(c, app); err != nil {
			return stdcli.Error(err)
		}

		stdcli.OK()
	}

	return nil
}

func cmdAppDelete(c *cli.Context) error {
	stdcli.NeedHelp(c)
	stdcli.NeedArg(c, 1)

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
	stdcli.NeedHelp(c)

	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.Error(err)
	}

	// FIXME: we should accept only --app (i.e. as a flag) to be consistent with other commands
	if len(c.Args()) > 0 {
		stdcli.NeedArg(c, 1)
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
	info.Add("Generation", a.Generation)
	info.Add("Processes", stdcli.Default(strings.Join(ps, " "), "(none)"))
	info.Add("Endpoints", strings.Join(endpoints, "\n            "))

	info.Print()

	return nil
}

func cmdAppParams(c *cli.Context) error {
	stdcli.NeedHelp(c)
	stdcli.NeedArg(c, 0)

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
	stdcli.NeedHelp(c)
	// need at least one argument
	stdcli.NeedArg(c, -1)

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
		if strings.Contains(err.Error(), "No updates are to be performed") {
			return stdcli.Error(fmt.Errorf("No updates are to be performed"))
		}
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
