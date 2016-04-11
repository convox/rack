package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/convox/rack/cmd/convox/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "apps",
		Action:      cmdApps,
		Description: "list deployed apps",
		Subcommands: []cli.Command{
			{
				Name:        "create",
				Description: "create a new application",
				Usage:       "<name>",
				Action:      cmdAppCreate,
			},
			{
				Name:        "delete",
				Description: "delete an application",
				Usage:       "<name>",
				Action:      cmdAppDelete,
			},
			{
				Name:        "info",
				Description: "see info about an app",
				Usage:       "[name]",
				Action:      cmdAppInfo,
				Flags:       []cli.Flag{appFlag},
			},
			{
				Name:        "params",
				Description: "list advanced parameters for an app",
				Usage:       "[name]",
				Action:      cmdAppParams,
				Flags:       []cli.Flag{appFlag},
				Subcommands: []cli.Command{
					{
						Name:        "set",
						Description: "update advanced parameters for an app",
						Usage:       "NAME=VALUE [NAME=VALUE]",
						Action:      cmdAppParamsSet,
						Flags:       []cli.Flag{appFlag},
					},
				},
			},
		},
	})
}

func cmdApps(c *cli.Context) {
	apps, err := rackClient(c).GetApps()

	if err != nil {
		stdcli.Error(err)
		return
	}

	t := stdcli.NewTable("APP", "STATUS")

	for _, app := range apps {
		t.AddRow(app.Name, app.Status)
	}

	t.Print()
}

func cmdAppCreate(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	if len(c.Args()) > 0 {
		app = c.Args()[0]
	}

	if app == "" {
		stdcli.Error(fmt.Errorf("must specify an app name"))
		return
	}

	fmt.Printf("Creating app %s... ", app)

	_, err = rackClient(c).CreateApp(app)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Println("CREATING")
}

func cmdAppDelete(c *cli.Context) {
	if len(c.Args()) < 1 {
		stdcli.Usage(c, "delete")
		return
	}

	app := c.Args()[0]

	fmt.Printf("Deleting %s... ", app)

	_, err := rackClient(c).DeleteApp(app)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Println("DELETING")
}

func cmdAppInfo(c *cli.Context) {
	var app string
	var err error

	if len(c.Args()) > 0 {
		app = c.Args()[0]
	} else {
		_, app, err = stdcli.DirApp(c, ".")
	}

	a, err := rackClient(c).GetApp(app)

	if err != nil {
		stdcli.Error(err)
		return
	}

	formation, err := rackClient(c).ListFormation(app)

	if err != nil {
		stdcli.Error(err)
		return
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
}

func cmdAppParams(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	params, err := rackClient(c).ListParameters(app)

	if err != nil {
		stdcli.Error(err)
		return
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
}

func cmdAppParamsSet(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	params := map[string]string{}

	for _, arg := range c.Args() {
		parts := strings.SplitN(arg, "=", 2)

		if len(parts) != 2 {
			stdcli.Error(fmt.Errorf("invalid argument: %s", arg))
			return
		}

		params[parts[0]] = parts[1]
	}

	fmt.Print("Updating parameters... ")

	err = rackClient(c).SetParameters(app, params)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Println("OK")
}
