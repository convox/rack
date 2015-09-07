package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
)

type App struct {
	Name string

	Status     string
	Repository string

	Outputs    map[string]string
	Parameters map[string]string
	Tags       map[string]string
}

type Apps []App

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "apps",
		Action:      cmdApps,
		Description: "list deployed apps",
		Subcommands: []cli.Command{
			{
				Name:        "create",
				Description: "create a new application",
				Usage:       "[name]",
				Action:      cmdAppCreate,
			},
			{
				Name:        "delete",
				Description: "delete an application",
				Usage:       "[name]",
				Action:      cmdAppDelete,
			},
			{
				Name:        "info",
				Description: "see info about an app",
				Usage:       "",
				Action:      cmdAppInfo,
				Flags:       []cli.Flag{appFlag},
			},
		},
	})
}

func cmdApps(c *cli.Context) {
	apps, err := rackClient().GetApps()

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
		fmt.Printf("Creating app... ")
	} else {
		fmt.Printf("Creating app %s... ", app)
	}

	a, err := rackClient().CreateApp(app)

	if err != nil {
		stdcli.Error(err)
		return
	}

	// poll for complete
	for {
		app, err := rackClient().GetApp(a.Name)

		if err != nil {
			stdcli.Error(err)
			return
		}

		if app.Status == "running" {
			break
		}

		time.Sleep(3 * time.Second)
	}

	if app == "" {
		fmt.Printf("OK, %s\n", a.Name)
	} else {
		fmt.Println("OK")
	}
}

func cmdAppDelete(c *cli.Context) {
	if len(c.Args()) < 1 {
		stdcli.Usage(c, "delete")
		return
	}

	app := c.Args()[0]

	fmt.Printf("Deleting %s... ", app)

	_, err := rackClient().DeleteApp(app)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Println("DELETING")
}

func cmdAppInfo(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")

	a, err := rackClient().GetApp(app)

	if err != nil {
		stdcli.Error(err)
		return
	}

	ps, err := rackClient().GetProcesses(app)

	if err != nil {
		stdcli.Error(err)
		return
	}

	pss := make([]string, len(ps))

	for i, p := range ps {
		pss[i] = p.Name
	}

	sort.Strings(pss)

	ports := []string{}

	for _, p := range ps {
		for _, port := range p.Ports {
			ports = append(ports, fmt.Sprintf("%s:%d", p.Name, port))
		}
	}

	sort.Strings(ports)

	fmt.Printf("Name       %s\n", a.Name)
	fmt.Printf("Status     %s\n", a.Status)
	fmt.Printf("Release    %s\n", stdcli.Default(a.Release, "(none)"))
	fmt.Printf("Processes  %s\n", stdcli.Default(strings.Join(pss, " "), "(none)"))

	if a.Balancer != "" {
		fmt.Printf("Hostname   %s\n", a.Balancer)
		fmt.Printf("Ports      %s\n", stdcli.Default(strings.Join(ports, " "), "(none)"))
	}
}
