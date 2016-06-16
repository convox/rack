package main

import (
	"fmt"
	"sort"
	"strings"

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
				Name:        "create",
				Description: "create a new application",
				Usage:       "<name>",
				Action:      cmdAppCreate,
				Flags:       []cli.Flag{rackFlag},
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
		return stdcli.ExitError(fmt.Errorf("`convox apps` does not take arguments. Perhaps you meant `convox apps create`?"))
	}

	if c.Bool("help") {
		stdcli.Usage(c, "")
		return nil
	}

	apps, err := rackClient(c).GetApps()
	if err != nil {
		return stdcli.ExitError(err)
	}

	t := stdcli.NewTable("APP", "STATUS")

	for _, app := range apps {
		t.AddRow(app.Name, app.Status)
	}

	t.Print()
	return nil
}

func cmdAppCreate(c *cli.Context) error {
	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.ExitError(err)
	}

	if len(c.Args()) > 0 {
		app = c.Args()[0]
	}

	if app == "" {
		return stdcli.ExitError(fmt.Errorf("must specify an app name"))
	}

	fmt.Printf("Creating app %s... ", app)

	_, err = rackClient(c).CreateApp(app)
	if err != nil {
		return stdcli.ExitError(err)
	}

	fmt.Println("CREATING")
	return nil
}

func cmdAppDelete(c *cli.Context) error {
	if len(c.Args()) < 1 {
		stdcli.Usage(c, "delete")
		return nil
	}

	app := c.Args()[0]

	fmt.Printf("Deleting %s... ", app)

	_, err := rackClient(c).DeleteApp(app)
	if err != nil {
		return stdcli.ExitError(err)
	}

	fmt.Println("DELETING")
	return nil
}

func cmdAppInfo(c *cli.Context) error {
	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.ExitError(err)
	}

	if len(c.Args()) > 0 {
		app = c.Args()[0]
	}

	a, err := rackClient(c).GetApp(app)
	if err != nil {
		return stdcli.ExitError(err)
	}

	formation, err := rackClient(c).ListFormation(app)
	if err != nil {
		return stdcli.ExitError(err)
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

	fmt.Printf("Name       %s\n", a.Name)
	fmt.Printf("Status     %s\n", a.Status)
	fmt.Printf("Release    %s\n", stdcli.Default(a.Release, "(none)"))
	fmt.Printf("Processes  %s\n", stdcli.Default(strings.Join(ps, " "), "(none)"))
	fmt.Printf("Endpoints  %s\n", strings.Join(endpoints, "\n           "))
	return nil
}

func cmdAppParams(c *cli.Context) error {
	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.ExitError(err)
	}

	params, err := rackClient(c).ListParameters(app)
	if err != nil {
		return stdcli.ExitError(err)
	}

	keys := []string{}

	for key, _ := range params {
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
		return stdcli.ExitError(err)
	}

	params := map[string]string{}

	for _, arg := range c.Args() {
		parts := strings.SplitN(arg, "=", 2)

		if len(parts) != 2 {
			return stdcli.ExitError(fmt.Errorf("invalid argument: %s", arg))
		}

		params[parts[0]] = parts[1]
	}

	fmt.Print("Updating parameters... ")

	err = rackClient(c).SetParameters(app, params)
	if err != nil {
		return stdcli.ExitError(err)
	}

	fmt.Println("OK")
	return nil
}
