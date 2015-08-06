package main

import (
	"encoding/json"
	"fmt"
	"net/url"
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
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "app",
				Usage: "app name. If not specified, use current directory.",
			},
		},
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
				Usage:       "<name>",
				Action:      cmdAppDelete,
			},
		},
	})
}

func cmdApps(c *cli.Context) {
	apps := getApps()

	longest := 3

	for _, app := range *apps {
		if len(app.Name) > longest {
			longest = len(app.Name)
		}
	}

	fmt.Printf(fmt.Sprintf("%%-%ds  STATUS\n", longest), "APP")

	for _, app := range *apps {
		fmt.Printf(fmt.Sprintf("%%-%ds  %%s\n", longest), app.Name, app.Status)
	}
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

	v := url.Values{}
	v.Set("name", app)
	data, err := ConvoxPostForm("/apps", v)

	if err != nil {
		stdcli.Error(err)
		return
	}

	data, err = ConvoxGet("/apps/" + app)

	if err != nil {
		stdcli.Error(err)
		return
	}

	var a *App
	err = json.Unmarshal(data, &a)

	if err != nil {
		stdcli.Error(err)
		return
	}

	// poll for complete
	for {
		data, err = ConvoxGet(fmt.Sprintf("/apps/%s/status", app))

		if err != nil {
			stdcli.Error(err)
			return
		}

		if string(data) == "running" {
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

	_, err := ConvoxDelete(fmt.Sprintf("/apps/%s", app))

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Println("OK - DELETE STARTED")
}

func getApps() *Apps {
	data, err := ConvoxGet("/apps")

	if err != nil {
		stdcli.Error(err)
		return nil
	}

	var apps *Apps
	err = json.Unmarshal(data, &apps)

	if err != nil {
		stdcli.Error(err)
		return nil
	}

	return apps
}
